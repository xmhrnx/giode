package main

import (
	"embed"

	"github.com/xmhrnx/giode"
)

// Giode ships no icon set: an app brings the icons it uses. These are
// embedded into the binary and registered by file name ("icons/star.svg"
// becomes "star"), so they can be looked up anywhere with giode.Icon.
//
//go:embed icons/*.svg
var iconFiles embed.FS

func init() {
	if err := giode.RegisterIconsFS(iconFiles, "icons"); err != nil {
		panic(err)
	}
}

// docIcon is a document parsed directly, without the registry.
var docIcon = giode.MustLoadIconSVG([]byte(`<svg viewBox="0 0 24 24">
	<rect x="2" y="2" width="8" height="8" rx="2"/>
	<circle cx="18" cy="18" r="4"/>
</svg>`))

// IconsShowcase is stateless: icons are stateless elements, so it can
// be built fresh every frame like Text.
func IconsShowcase() giode.Element {
	size := func(c string) giode.Styles { return giode.Styles{Color: c, Width: 22, Height: 22} }
	badge, _ := giode.LookupIcon("badge")
	return giode.Stack(boxStyle,
		giode.Text("Icons", textStyle),
		giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter},
			giode.Icon("star", size("#facc15")),
			giode.Icon("heart", size("#ef4444")),
			giode.Icon("trash", size("#94a3b8")),
			giode.Icon("bell", size("white")),   // stroke-based
			giode.Icon("home", size("#a78bfa")), // stroke-based
			docIcon.Icon(size("#22d3ee")),
			badge.(*giode.IconDocument).IconColored(giode.Styles{Width: 22}), // keeps the file's colors
		),
	)
}
