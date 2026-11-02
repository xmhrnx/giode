// Package licenses shows the open-source licenses of the modules an app
// is built from: the notices screen app stores and licenses ask for.
// The giode CLI writes the data (giode licenses):
//
//	//go:embed assets/licenses.json
//	var licensesJSON []byte
//
//	view := licenses.New(licensesJSON)
package licenses

import (
	"encoding/json"

	"gioui.org/layout"

	"github.com/xmhrnx/giode/components/lazylist"
	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/i18n"
	"github.com/xmhrnx/giode/styles"
	"github.com/xmhrnx/giode/styles/properties"
)

// License is a module's license, as giode licenses writes it.
type License struct {
	Module  string `json:"module"`
	Version string `json:"version,omitempty"`
	Type    string `json:"type,omitempty"`
	Text    string `json:"text"`
}

// View is a list of licenses: a row per module with its license type,
// expanding to the license text on a tap.
type View struct {
	licenses []License
	open     map[string]bool
	rows     map[string]*elements.Pressable
	list     *lazylist.List
	st       styles.Styles
}

// New returns a View of the licenses in data, the JSON giode licenses
// writes. The styles are optional: Color, FontSize and Background apply.
func New(data []byte, st ...styles.Styles) *View {
	v := &View{open: map[string]bool{}, rows: map[string]*elements.Pressable{}}
	json.Unmarshal(data, &v.licenses)
	if len(st) > 0 {
		v.st = st[0]
	}
	v.list = lazylist.New(
		func() int { return len(v.licenses) },
		func(i int) any { return v.licenses[i].Module },
		v.row,
		styles.Styles{Background: v.st.Background},
	).Version(func(i int) any { return v.open[v.licenses[i].Module] })
	return v
}

// Licenses returns the licenses shown.
func (v *View) Licenses() []License {
	return v.licenses
}

func (v *View) row(i int) elements.Element {
	l := v.licenses[i]
	color := v.st.Color
	if color == "" {
		color = "theme.text"
	}
	size := v.st.FontSize
	if size == 0 {
		size = 15
	}
	head, ok := v.rows[l.Module]
	if !ok {
		head = elements.NewPressable(func(p elements.PressState) elements.Element {
			title := l.Module
			if l.Version != "" {
				title += " " + l.Version
			}
			kind := l.Type
			if kind == "" {
				kind = i18n.T("giode.license")
			}
			bg := properties.Color("")
			if p.Hovered || p.Pressed {
				bg = "theme.surface"
			}
			return elements.Box(styles.Styles{Width: -1, Padding: properties.SymmetricInset(16, 12), Background: bg,
				FlexDirection: properties.FlexDirectionColumn, Gap: 2},
				elements.Text(title, styles.Styles{Color: color, FontSize: size, FontWeight: properties.FontWeightSemiBold}),
				elements.Text(kind, styles.Styles{Color: properties.Color(string(color) + "/0.6"), FontSize: size - 2}),
			)
		}).Label(l.Module).OnClick(func(elements.ClickEvent) {
			v.open[l.Module] = !v.open[l.Module]
		})
		v.rows[l.Module] = head
	}
	if !v.open[l.Module] {
		return head
	}
	text := l.Text
	if text == "" {
		text = i18n.T("giode.no_license")
	}
	return elements.Box(styles.Styles{Width: -1, FlexDirection: properties.FlexDirectionColumn},
		head,
		elements.Box(styles.Styles{Width: -1, Padding: properties.Inset{Left: 16, Right: 16, Bottom: 16}},
			elements.Text(text, styles.Styles{Color: color, FontSize: size - 3, FontFamily: "Go Mono"}).Selectable("giode.license."+l.Module)),
	)
}

// Layout lays the list out.
func (v *View) Layout(gtx layout.Context) layout.Dimensions {
	return v.list.Layout(gtx)
}
