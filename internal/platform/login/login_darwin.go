//go:build darwin && !ios

package login

import (
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmhrnx/giode/internal/platform/sandbox"
)

func launchAgentFile(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", name+".login.plist"), nil
}

func Set(e Entry, on bool) error {
	if sandbox.Enabled() {
		// A sandboxed app can't write a LaunchAgent; App Review rejects
		// one that tries. It's a login item instead (macOS 13 and later).
		return sandbox.SetLoginItem(on)
	}
	name := e.Name
	path, err := launchAgentFile(name)
	if err != nil {
		return err
	}
	if !on {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	cmd, err := Command()
	if err != nil {
		return err
	}
	// An app bundle opens through Launch Services, as the user opening
	// it would, rather than as a bare process.
	if i := strings.Index(cmd[0], ".app/Contents/MacOS/"); i >= 0 {
		cmd = []string{"/usr/bin/open", "-a", cmd[0][:i+len(".app")], "--args", Arg}
	}
	var args strings.Builder
	for _, c := range cmd {
		args.WriteString("\n\t\t<string>" + html.EscapeString(c) + "</string>")
	}
	// Login Items in System Settings lists the agent under the app
	// (its name and icon) rather than under /usr/bin/open.
	associated := ""
	if e.ID != "" {
		associated = "\n\t<key>AssociatedBundleIdentifiers</key>\n\t<string>" + html.EscapeString(e.ID) + "</string>"
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + html.EscapeString(name) + `.login</string>
	<key>ProgramArguments</key>
	<array>` + args.String() + `
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>` + associated + `
</dict>
</plist>
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(plist), 0o644)
}

func Enabled(name string) bool {
	if sandbox.Enabled() {
		return sandbox.LoginItemEnabled()
	}
	path, err := launchAgentFile(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func ByStartupTask() bool { return false }
