package host

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xmhrnx/giode/update"
)

// The single-instance lock is a file in the temporary directory holding
// the owner's pid, the loopback port it listens on and a random token.
// A second instance hands its command-line arguments to the owner over
// that port, proving it can read the file (which only the user can)
// with the token.

// InstanceLock is the single-instance lock of an application id, held
// by the first instance to start. Get it with LockInstance.
type InstanceLock struct {
	path    string
	running bool
	ln      net.Listener
	token   string
}

// LockInstance claims the single-instance lock for id (the app's id,
// "com.example.player"). When another live instance holds it, Running
// reports true: hand it this launch's arguments with Forward and exit.
// Otherwise this instance holds the lock: Serve receives later
// instances' arguments, and Release frees the lock on exit. An app
// restarted by an update first waits for the old process to exit.
func LockInstance(id string) *InstanceLock {
	second, path, owner := acquireInstanceLock(id)
	if owner == nil {
		owner = &InstanceLock{}
	}
	owner.path, owner.running = path, second
	return owner
}

// Running reports whether another instance holds the lock.
func (l *InstanceLock) Running() bool { return l.running }

// Forward hands args to the instance holding the lock, which gets them
// through its Serve function.
func (l *InstanceLock) Forward(args []string) error {
	return forwardLaunch(l.path, args)
}

// Serve calls fn, on its own goroutine, with the arguments of each
// later instance that forwards its launch. It does nothing when another
// instance holds the lock.
func (l *InstanceLock) Serve(fn func(args []string)) {
	if l.running {
		return
	}
	l.serveLaunches(fn)
}

// Release frees a lock this instance holds.
func (l *InstanceLock) Release() {
	if l.running {
		return
	}
	releaseInstanceLock(l.path, l)
}

// acquireInstanceLock claims a single-instance lock for the given
// application id. It reports whether another live instance already
// holds the lock.
func acquireInstanceLock(id string) (second bool, path string, owner *InstanceLock) {
	WaitForUpdate()
	if id == "" {
		id = "giode"
	}
	path = filepath.Join(os.TempDir(), "giode-"+sanitizeID(id)+".lock")

	// The lock owner writes its pid; a stale file (dead process or
	// one that crashed before writing) is reclaimed.
	for attempt := 0; attempt < 5; attempt++ {
		data, err := os.ReadFile(path)
		if err != nil {
			break
		}
		pid, _, _ := parseLock(data)
		if pid > 0 && processAlive(pid) {
			return true, path, nil
		}
		// Empty or dead: give a racing writer a moment to fill in its
		// pid before reclaiming.
		if attempt < 4 {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		os.Remove(path)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Lost the race with another instance.
		return true, path, nil
	}
	defer f.Close()
	owner = &InstanceLock{}
	port := 0
	if ln, err := net.Listen("tcp", "127.0.0.1:0"); err == nil {
		var tok [16]byte
		rand.Read(tok[:])
		owner.ln, owner.token = ln, hex.EncodeToString(tok[:])
		port = ln.Addr().(*net.TCPAddr).Port
	}
	fmt.Fprintf(f, "%d\n%d\n%s\n", os.Getpid(), port, owner.token)
	return false, path, owner
}

// WaitForUpdate waits, in an app restarted by an update (see the
// update package), for the old process to exit, so the new one takes
// over its files and lock. LockInstance calls it; an app without the
// lock calls it early in main.
func WaitForUpdate() {
	pid, err := strconv.Atoi(os.Getenv(update.WaitEnv))
	if err != nil {
		if pid = TakeWaitArg(); pid == 0 {
			return
		}
	}
	os.Unsetenv(update.WaitEnv)
	for deadline := time.Now().Add(15 * time.Second); processAlive(pid) && time.Now().Before(deadline); {
		time.Sleep(50 * time.Millisecond)
	}
	update.Cleanup()
}

// WaitArg is the argument an app restarted with other rights (see
// giode's App.RestartElevated) gets before its own, naming the old
// process to wait for: elevation (UAC, pkexec, an administrator prompt)
// doesn't pass the environment on.
const WaitArg = "--giode-wait-pid="

// TakeWaitArg removes WaitArg from os.Args and returns its pid, or 0.
func TakeWaitArg() int {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], WaitArg) {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimPrefix(os.Args[1], WaitArg))
	os.Args = append(os.Args[:1], os.Args[2:]...)
	return pid
}

// parseLock reads a lock file: pid, then port and token.
func parseLock(data []byte) (pid, port int, token string) {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	pid, _ = strconv.Atoi(strings.TrimSpace(lines[0]))
	if len(lines) >= 3 {
		port, _ = strconv.Atoi(strings.TrimSpace(lines[1]))
		token = strings.TrimSpace(lines[2])
	}
	return pid, port, token
}

// releaseInstanceLock removes the lock file of the owning instance.
func releaseInstanceLock(path string, owner *InstanceLock) {
	if owner != nil && owner.ln != nil {
		owner.ln.Close()
	}
	if path != "" {
		os.Remove(path)
	}
}

// launchMessage is what a second instance sends the owner.
type launchMessage struct {
	Token string   `json:"token"`
	Args  []string `json:"args"`
}

// forwardLaunch sends args to the instance holding the lock at path.
func forwardLaunch(path string, args []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, port, token := parseLock(data)
	if port == 0 {
		return fmt.Errorf("giode: the running instance doesn't take launches")
	}
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 2*time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	if err := json.NewEncoder(c).Encode(launchMessage{Token: token, Args: args}); err != nil {
		return err
	}
	reply, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(reply) != "ok" {
		return fmt.Errorf("giode: the running instance refused the launch")
	}
	return nil
}

// serveLaunches hands launches from second instances to handle.
func (l *InstanceLock) serveLaunches(handle func(args []string)) {
	if l == nil || l.ln == nil {
		return
	}
	go func() {
		for {
			c, err := l.ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				c.SetDeadline(time.Now().Add(5 * time.Second))
				var m launchMessage
				if err := json.NewDecoder(io.LimitReader(c, 1<<20)).Decode(&m); err != nil {
					return
				}
				if subtle.ConstantTimeCompare([]byte(m.Token), []byte(l.token)) != 1 {
					io.WriteString(c, "no\n")
					return
				}
				io.WriteString(c, "ok\n")
				handle(m.Args)
			}()
		}
	}()
}

func sanitizeID(id string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, id)
}
