//go:build windows

package giode

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Explorer's classic context menu reads verbs from these keys under
// HKEY_CURRENT_USER\Software\Classes, which needs no administrator
// rights. The keys created are recorded under Software\<prefix> so
// uninstalling finds them.

const contextActionsRecord = "GiodeContextActions"

func installContextActions(prefix, exe string, actions []ContextAction) error {
	uninstallContextActions(prefix)
	var created []string
	add := func(parent, id, label, arg string) error {
		path := `Software\Classes\` + parent + `\shell\` + prefix + "." + id
		k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		k.SetStringValue("MUIVerb", label)
		k.SetStringValue("Icon", exe)
		k.Close()
		if arg == "%1" {
			// Files and folders: Explorer drops the whole selection on
			// the app's COM class (see contextserver_windows.go) rather
			// than starting a copy per item with the command below,
			// which stays for programs that only run commands.
			clsid := ContextActionCLSID(prefix, id)
			if err := registerDropTarget(path, clsid, exe); err != nil {
				return err
			}
			created = append(created, `Software\Classes\CLSID\`+clsid)
		}
		c, _, err := registry.CreateKey(registry.CURRENT_USER, path+`\command`, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		defer c.Close()
		created = append(created, path)
		return c.SetStringValue("", `"`+exe+`" `+contextActionFlag+id+` "`+arg+`"`)
	}
	var errs []error
	for _, c := range actions {
		if c.onFiles() {
			if len(c.Extensions) == 0 {
				errs = append(errs, add("*", c.ID, c.Label, "%1"))
			}
			for _, ext := range c.Extensions {
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				errs = append(errs, add(`SystemFileAssociations\`+ext, c.ID, c.Label, "%1"))
			}
		}
		if c.Folders {
			errs = append(errs, add("Directory", c.ID, c.Label, "%1"))
		}
		if c.FolderBackground {
			errs = append(errs, add(`Directory\Background`, c.ID, c.Label, "%V"))
		}
	}
	if k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\`+prefix, registry.ALL_ACCESS); err == nil {
		k.SetStringsValue(contextActionsRecord, created)
		k.Close()
	}
	return errors.Join(errs...)
}

// registerDropTarget makes the verb at path a DropTarget verb for the
// COM class clsid, served by exe.
func registerDropTarget(path, clsid, exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	k.SetStringValue("MultiSelectModel", "Player")
	k.Close()
	d, _, err := registry.CreateKey(registry.CURRENT_USER, path+`\DropTarget`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	d.SetStringValue("CLSID", clsid)
	d.Close()
	s, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\CLSID\`+clsid+`\LocalServer32`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.SetStringValue("", `"`+exe+`"`)
}

func uninstallContextActions(prefix string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\`+prefix, registry.ALL_ACCESS)
	if err != nil {
		return nil
	}
	defer k.Close()
	paths, _, _ := k.GetStringsValue(contextActionsRecord)
	var errs []error
	for _, p := range paths {
		if !strings.Contains(p, `\shell\`+prefix+".") && !isOurCLSID(prefix, p) {
			continue // not ours
		}
		errs = append(errs, deleteKeyTree(registry.CURRENT_USER, p))
	}
	k.DeleteValue(contextActionsRecord)
	return errors.Join(errs...)
}

// deleteKeyTree deletes a key and its subkeys.
func deleteKeyTree(root registry.Key, path string) error {
	k, err := registry.OpenKey(root, path, registry.ALL_ACCESS)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	subs, _ := k.ReadSubKeyNames(-1)
	k.Close()
	for _, s := range subs {
		if err := deleteKeyTree(root, path+`\`+s); err != nil {
			return err
		}
	}
	return registry.DeleteKey(root, path)
}

// isOurCLSID reports whether path is the COM class key of one of the
// app's context actions.
func isOurCLSID(prefix, path string) bool {
	const dir = `Software\Classes\CLSID\`
	return strings.HasPrefix(path, dir) && strings.HasPrefix(strings.TrimPrefix(path, dir), "{")
}
