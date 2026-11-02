// Command workbench exercises the interactive building blocks
// together: browse folders in a lazily loaded TreeView, drop files onto
// the window to list them in a VirtualList, click rows (Pressable,
// Keyed state), copy a path (clipboard), and get notified with toasts.
package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/xmhrnx/giode"
)

// folder is a TreeView node whose children load on first expansion.
type folder struct {
	path     string
	isDir    bool
	loaded   bool
	children []giode.TreeNode
}

func (f *folder) Key() any                   { return f.path }
func (f *folder) Children() []giode.TreeNode { return f.children }
func (f *folder) HasChildren() bool          { return f.isDir && (!f.loaded || len(f.children) > 0) }
func (f *folder) Label() string              { return filepath.Base(f.path) }

// load reads the folder on a background goroutine and applies the
// result on the UI goroutine with Post.
func (f *folder) load(app *giode.App) {
	if f.loaded || !f.isDir {
		return
	}
	go func() {
		time.Sleep(300 * time.Millisecond) // let the loading state show
		entries, _ := os.ReadDir(f.path)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})
		var kids []giode.TreeNode
		for _, e := range entries {
			if e.Name()[0] == '.' {
				continue
			}
			kids = append(kids, &folder{path: filepath.Join(f.path, e.Name()), isDir: e.IsDir()})
		}
		app.Post(func() { f.children, f.loaded = kids, true })
	}()
}

type model struct {
	files    []string
	selected int
}

func main() {
	giode.UseGofont()
	giode.SetMode(giode.ModeDark)

	m := &model{selected: -1}
	toasts := giode.Toaster()

	app := giode.New(giode.Config{
		Title:      "Workbench",
		Size:       image.Pt(760, 520),
		Resizable:  true,
		Background: "theme.background",
		Logger:     log.Default(),
		LogLevel:   giode.LogLevelInfo,
		// Runs on the UI goroutine, so the model needs no locks.
		OnFileDrop: func(paths []string) {
			m.files = append(m.files, paths...)
			toasts.Show(giode.ToastSuccess, fmt.Sprintf("Added %d file(s)", len(paths)))
		},
	})

	root, _ := os.Getwd()
	roots := []giode.TreeNode{&folder{path: root, isDir: true}}
	tree := giode.TreeView(func() []giode.TreeNode { return roots },
		func(n giode.TreeNode, s giode.TreeRowState) giode.Element {
			f := n.(*folder)
			col := "theme.text"
			if !f.isDir {
				col = "theme.muted"
			}
			return giode.Text(filepath.Base(f.path), giode.Styles{Color: col, FontSize: 13, TextOverflow: giode.TextOverflowEllipsis})
		}, giode.Styles{Height: 26}).
		OnExpand(func(n giode.TreeNode) { n.(*folder).load(app) }).
		OnActivate(func(n giode.TreeNode) {
			if f := n.(*folder); !f.isDir {
				m.files = append(m.files, f.path)
				toasts.Show(giode.ToastInfo, "Added "+filepath.Base(f.path))
			}
		})

	list := giode.VirtualList(func() int { return len(m.files) }, 34, func(i int) giode.Element {
		path := m.files[i]
		return giode.Keyed("files", path, func() *giode.PressableElement {
			return giode.Pressable(func(s giode.PressState) giode.Element {
				bg := ""
				switch {
				case m.selected >= 0 && m.selected < len(m.files) && m.files[m.selected] == path:
					bg = "theme.primary/0.25"
				case s.Hovered:
					bg = "theme.text/0.06"
				}
				return giode.Row(giode.Styles{Width: -1, Height: -1, Background: bg, Align: giode.AlignCenter,
					Padding: giode.SymmetricInset(12, 0), Gap: 8, Wrap: giode.WrapNoWrap},
					giode.Text(filepath.Base(path), giode.Styles{Color: "theme.text"}),
					giode.Text(filepath.Dir(path), giode.Styles{Color: "theme.muted", FontSize: 12, TextOverflow: giode.TextOverflowEllipsis}),
				)
			}).Label(filepath.Base(path)).OnClick(func(e giode.ClickEvent) {
				for j, f := range m.files {
					if f == path {
						m.selected = j
					}
				}
				if e.Clicks == 2 {
					toasts.Show(giode.ToastInfo, "Opened "+filepath.Base(path))
				}
			})
		})
	}, giode.Styles{Background: "theme.surface"})

	copyPath := giode.CopyButton(func() string {
		if m.selected >= 0 && m.selected < len(m.files) {
			return m.files[m.selected]
		}
		return ""
	}).Labels("Copy path", "Copied").OnCopy(func(string) {
		toasts.Show(giode.ToastInfo, "Path copied to the clipboard")
	})
	var scan *giode.ButtonElement
	scan = giode.Button("Rescan").Icon(giode.IconSVGStroke("M4 12a8 8 0 0 1 14-5.3M20 12a8 8 0 0 1-14 5.3M18 3v4h-4M6 21v-4h4", 2,
		giode.Styles{Width: 16})).OnClick(func() {
		scan.Loading(true)
		go func() {
			time.Sleep(1500 * time.Millisecond)
			app.Post(func() {
				scan.Loading(false)
				toasts.Show(giode.ToastSuccess, "Scan finished")
			})
		}()
	})
	clear := giode.Button("Clear").Variant(giode.VariantOutlined).OnClick(func() {
		n := len(m.files)
		old := m.files
		m.files, m.selected = nil, -1
		toasts.ShowAction(giode.ToastWarning, fmt.Sprintf("Cleared %d file(s)", n), "Undo", func() {
			m.files = old
		})
	})

	if err := app.Run(func() giode.Element {
		hint := "Drop files anywhere on this window"
		if len(m.files) > 0 {
			hint = fmt.Sprintf("%d file(s)", len(m.files))
		}
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(16), Gap: 12},
			giode.Row(giode.Styles{Width: -1, Gap: 8, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap},
				giode.Box(giode.Styles{FlexGrow: 1}, giode.Text(hint, giode.Styles{Color: "theme.text", FontSize: 16})),
				scan, copyPath, clear,
			),
			giode.Row(giode.Styles{Width: -1, FlexGrow: 1, Gap: 12, Wrap: giode.WrapNoWrap},
				giode.Box(giode.Styles{Width: 260, Height: -1, BorderRadius: 8, Overflow: giode.OverflowHidden,
					BorderWidth: 1, BorderColor: "theme.border", Background: "theme.surface"}, tree),
				giode.Box(giode.Styles{FlexGrow: 1, Height: -1, BorderRadius: 8, Overflow: giode.OverflowHidden,
					BorderWidth: 1, BorderColor: "theme.border"}, list),
			),
			toasts,
		)
	}); err != nil {
		log.Fatal(err)
	}
}
