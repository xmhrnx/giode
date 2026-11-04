package host

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestSingleInstance_ForwardsLaunch(t *testing.T) {
	id := "giode-test-" + filepath.Base(t.TempDir())
	second, path, owner := acquireInstanceLock(id)
	if second || owner == nil {
		t.Fatal("first instance didn't get the lock")
	}
	defer releaseInstanceLock(path, owner)
	// Windows has no permission bits; its temporary directory is the
	// user's own.
	if info, err := os.Stat(path); runtime.GOOS != "windows" && (err != nil || info.Mode().Perm()&0o077 != 0) {
		t.Errorf("lock file mode %v (%v), want private", info.Mode(), err)
	}
	got := make(chan []string, 1)
	owner.serveLaunches(func(args []string) { got <- args })

	second, path2, owner2 := acquireInstanceLock(id)
	if !second || owner2 != nil || path2 != path {
		t.Fatal("second instance got the lock")
	}
	if err := forwardLaunch(path2, []string{"--giode-action=ask", "/tmp/x.txt"}); err != nil {
		t.Fatal(err)
	}
	select {
	case args := <-got:
		if !reflect.DeepEqual(args, []string{"--giode-action=ask", "/tmp/x.txt"}) {
			t.Errorf("forwarded %q", args)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the launch never arrived")
	}

	// A wrong token is refused.
	data, _ := os.ReadFile(path)
	pid, port, _ := parseLock(data)
	bad := filepath.Join(t.TempDir(), "bad.lock")
	os.WriteFile(bad, []byte(fmtLock(pid, port, "nope")), 0o600)
	if err := forwardLaunch(bad, nil); err == nil {
		t.Error("a launch with the wrong token was accepted")
	}
}

func fmtLock(pid, port int, token string) string {
	return strconv.Itoa(pid) + "\n" + strconv.Itoa(port) + "\n" + token + "\n"
}

// TestLockInstance checks the exported API: the first instance holds
// the lock and serves, the second forwards.
func TestLockInstance(t *testing.T) {
	id := "giode-test-api-" + filepath.Base(t.TempDir())
	first := LockInstance(id)
	if first.Running() {
		t.Fatal("first instance sees another running")
	}
	defer first.Release()
	got := make(chan []string, 1)
	first.Serve(func(args []string) { got <- args })
	second := LockInstance(id)
	if !second.Running() {
		t.Fatal("second instance got the lock")
	}
	second.Release() // must not free the first's lock
	if err := second.Forward([]string{"movie.mkv"}); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-got:
		if len(a) != 1 || a[0] != "movie.mkv" {
			t.Errorf("%q", a)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no launch")
	}
}

func TestTakeWaitArg(t *testing.T) {
	saved := os.Args
	defer func() { os.Args = saved }()
	os.Args = []string{"app", WaitArg + "1234", "file.img"}
	if pid := TakeWaitArg(); pid != 1234 {
		t.Errorf("pid %d, want 1234", pid)
	}
	if len(os.Args) != 2 || os.Args[1] != "file.img" {
		t.Errorf("os.Args = %q, want the wait argument removed", os.Args)
	}
	if pid := TakeWaitArg(); pid != 0 {
		t.Errorf("second call gave %d", pid)
	}
}

func TestLockInstance_UnwritableTempDir(t *testing.T) {
	// A temporary directory the lock can't be written in (Android's
	// /data/local/tmp, for apps) mustn't read as another instance.
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	l := LockInstance("com.example.unwritable")
	if l.Running() {
		t.Fatal("a lock that couldn't be written was taken for another instance")
	}
	l.Serve(func([]string) {})
	l.Release()
}

// Copies started together (Explorer starts one per selected file) find
// the lock file the winner has created but not yet filled in: they
// treat it as held and wait for the port rather than reclaiming it.
func TestLockInstance_StartingOwner(t *testing.T) {
	id := "giode-test-" + filepath.Base(t.TempDir())
	path := filepath.Join(os.TempDir(), "giode-"+sanitizeID(id)+".lock")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	second, _, owner := acquireInstanceLock(id)
	if !second || owner != nil {
		t.Fatal("a just-created empty lock was reclaimed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("the starting owner's lock file was removed")
	}

	// The owner finishes starting while the second copy waits.
	got := make(chan []string, 1)
	_, realPath, real := acquireInstanceLock(id + "-owner")
	defer releaseInstanceLock(realPath, real)
	real.serveLaunches(func(args []string) { got <- args })
	go func() {
		time.Sleep(300 * time.Millisecond)
		data, _ := os.ReadFile(realPath)
		os.WriteFile(path, data, 0o600)
	}()
	if err := forwardLaunch(path, []string{"a.txt"}); err != nil {
		t.Fatal(err)
	}
	select {
	case args := <-got:
		if !reflect.DeepEqual(args, []string{"a.txt"}) {
			t.Errorf("forwarded %q", args)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the launch wasn't forwarded")
	}

	// An empty lock file left by a crash long ago is reclaimed.
	old := time.Now().Add(-time.Minute)
	os.WriteFile(path, nil, 0o600)
	os.Chtimes(path, old, old)
	second, p, o := acquireInstanceLock(id)
	if second || o == nil {
		t.Fatal("a stale empty lock wasn't reclaimed")
	}
	releaseInstanceLock(p, o)
}
