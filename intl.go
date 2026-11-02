package giode

import (
	"io/fs"

	"github.com/xmhrnx/giode/i18n"
)

// IntlOptions turns on translations (Config.Intl): message files named
// by locale - locales/en.json, locales/fa.json, locales/pt-BR.json -
// each a JSON object of keys to text with {placeholders}, or to plural
// forms:
//
//	{
//		"greeting": "Hello, {name}!",
//		"unread": {"one": "{count} message", "other": "{count} messages"}
//	}
type IntlOptions struct {
	// Dir is the folder holding the message files, "locales" by
	// default. A relative folder is looked for in the working
	// directory, next to the executable, then in a macOS app bundle's
	// Resources folder. With FS, it's a folder inside FS.
	Dir string
	// FS, when set, is read instead of the disk: an embedded folder
	// (//go:embed locales), which ships inside the executable and so
	// also works on Android, iOS and the web, where apps have no
	// folder beside them.
	FS fs.FS
	// Fallback is the locale used for keys the current one lacks,
	// "en" by default.
	Fallback string
}

func (o *IntlOptions) load() error {
	dir := o.Dir
	if dir == "" {
		dir = "locales"
	}
	if o.Fallback != "" {
		i18n.SetFallback(o.Fallback)
	}
	if o.FS != nil {
		return i18n.Load(o.FS, dir)
	}
	return i18n.LoadDir(dir)
}
