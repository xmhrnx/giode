//go:build darwin && !ios

package giode

import (
	"html"
	"os"
	"path/filepath"
	"strings"
)

func launchAgentFile(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", name+".login.plist"), nil
}

func setLaunchAtLogin(name string, on bool) error {
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
	cmd, err := loginCommand()
	if err != nil {
		return err
	}
	// An app bundle opens through Launch Services, as the user opening
	// it would, rather than as a bare process.
	if i := strings.Index(cmd[0], ".app/Contents/MacOS/"); i >= 0 {
		cmd = []string{"/usr/bin/open", "-a", cmd[0][:i+len(".app")], "--args", loginArg}
	}
	var args strings.Builder
	for _, c := range cmd {
		args.WriteString("\n\t\t<string>" + html.EscapeString(c) + "</string>")
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
	<string>Interactive</string>
</dict>
</plist>
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(plist), 0o644)
}

func launchAtLogin(name string) bool {
	path, err := launchAgentFile(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func launchedByStartupTask() bool { return false }
