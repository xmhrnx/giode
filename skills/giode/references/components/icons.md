# Icons

Giode ships **no icon set**. Your app brings the icons it uses — from any set, under the
license you choose — and only those end up in the binary. Icons are stateless elements:
size comes from `Width`/`Height` (in dp, default 24; set one for a square icon), color
from `Color` (default black).

## Registering icons by name

Embed a folder of `.svg` files and register it once; every file becomes a named icon
(`icons/home.svg` → `"home"`, `icons/brand/github.svg` → `"brand/github"`):

```go
//go:embed icons
var iconFS embed.FS

func init() {
	if err := giode.RegisterIconsFS(iconFS, "icons"); err != nil {
		log.Fatal(err)
	}
}

giode.Icon("home", giode.Styles{Color: "theme.text", Width: 20})
```

Other ways to fill the registry:

| Function | Source |
| --- | --- |
| `RegisterIconsFS(fsys, dir)` | every `.svg` under `dir` in an `fs.FS` (recursive) |
| `RegisterIconsDir(dir)` | every `.svg` under a directory on disk |
| `RegisterIconSVG(name, data)` | one SVG document's bytes |
| `RegisterIcon(name, src)` | any `IconSource`: a document, `IconPath{Data: "..."}`, an IconVG icon |

An unknown name never panics: it renders as empty space of the icon's size, and the
registry's `OnMissing` hook (on `icon.Default`) is called once per name — set it to log
typos. `LookupIcon(name)` returns the source, `IconNames()` lists what's registered, and
`icon.NewRegistry()` creates a separate registry (for libraries that must not collide with
the app's names).

## SVG documents

```go
doc, err := giode.LoadIconSVG(data)         // or MustLoadIconSVG, LoadIconSVGFile, LoadIconSVGFS
doc.Icon(giode.Styles{Color: "#fff"})        // single color: follows Color, like text
doc.IconColored(giode.Styles{Width: 32})     // keeps the file's own fill/stroke colors
```

Supported: `<path>`, `<rect>` (with `rx`/`ry`), `<circle>`, `<ellipse>`, `<line>`,
`<polyline>`, `<polygon>`, nested `<g>` groups; `fill`, `stroke`, `stroke-width`,
`stroke-linecap`/`linejoin` as attributes or in `style`, inherited as in SVG; `transform`
(`matrix`, `translate`, `scale`, `rotate`, `skewX`, `skewY`); `display="none"`. Both
fill-based sets (Material Symbols, Font Awesome, Bootstrap Icons) and stroke-based ones
(Lucide, Feather, Tabler, Heroicons outline) render. Gradients, masks, text and filters
are ignored. The drawing is scaled uniformly to fit its box and centered. Documents are
parsed once, so rendering costs no parsing.

## Path data

For sets distributed as path strings:

```go
giode.IconSVG("M3 6h18v2H3V6zm0 5h18v2H3v-2z", giode.Styles{Width: 24}) // 24x24 viewBox
giode.IconSVGCustom(d, image.Pt(512, 512), st)                         // custom viewBox
giode.IconSVGStroke(d, 2, st)                                          // outline icons
giode.RegisterIcon("menu", giode.IconPath{Data: d})
```

All SVG path commands are supported (`M L H V C S Q T A Z`, absolute and relative, compact
number syntax), including true elliptical arcs. Path strings are parsed once and cached.

## IconVG

Gio's native vector format:

```go
ic, err := giode.LoadIconVG(data) // decode once
ic.Icon(giode.Styles{Width: 24, Color: "white"})
giode.IconIconVG(data, st)        // convenience; decoding is cached, invalid data renders nothing
```

## Image

```go
giode.IconImage(img, giode.Styles{Width: 24, Height: 24})
```

## Icons in components

Components that show icons take an `IconSource`: `Menu.IconItem(src, label, action)`,
`Checkbox.Icon(src)`, `IconButton(element)`. Their built-in indicators (chevrons, the
check mark) are drawn shapes, not icons, so they need nothing loaded.
