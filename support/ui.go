package support

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"gioui.org/layout"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/components/checkbox"
	"github.com/xmhrnx/giode/components/input"
	"github.com/xmhrnx/giode/components/sheet"
	"github.com/xmhrnx/giode/crash"
	"github.com/xmhrnx/giode/store"
)

// Support is the report form and the tips sheet. Lay it out anywhere in
// the view: it takes no space and paints its sheets over the window.
type Support struct {
	app *giode.App
	o   Options

	reportOpen bool
	report     *sheet.Sheet
	title      *input.Input
	details    *input.Input
	email      *input.Input
	system     *checkbox.Checkbox
	withCrash  *checkbox.Checkbox
	withLogs   *checkbox.Checkbox
	send       *giode.ButtonElement
	cancel     *giode.ButtonElement
	crash      *crash.Report
	sending    bool
	problem    string

	tipsOpen bool
	tips     *sheet.Sheet
	loading  bool
	products []store.Product
	buttons  []*giode.ButtonElement
	tipError string

	toasts *giode.ToasterElement
	own    bool
}

// New returns the report form and the tips sheet for o.
func New(a *giode.App, o Options) *Support {
	if o.App == "" {
		o.App = giode.AppID
	}
	if o.Name == "" {
		o.Name = giode.AppName
	}
	s := &Support{app: a, o: o, toasts: o.Toasts}
	if s.toasts == nil {
		s.toasts, s.own = giode.Toaster(), true
	}

	s.title = input.New("A short summary").Label("Summary")
	s.details = input.New("What happened, and what did you expect?").Label("Details").Multiline(true).Lines(4, 10)
	s.email = input.New("Email, if you'd like a reply (optional)").Label("Email").Keyboard(input.KeyboardEmail)
	s.system = checkbox.New("Include the app version and system details")
	s.system.SetChecked(true)
	s.withCrash = checkbox.New("")
	s.withLogs = checkbox.New("Include recent logs")
	s.send = giode.Button("Send").OnClick(s.submit)
	s.cancel = giode.Button("Cancel").Variant(giode.VariantGhost).OnClick(func() { s.reportOpen = false })
	s.report = sheet.New(&s.reportOpen, s.reportBody).Title("Report a problem").Footer(func() giode.Element {
		return giode.Row(giode.Styles{Width: -1, Gap: 8, Justify: giode.JustifyEnd}, s.cancel, s.send)
	}).Adaptive(true)

	s.tips = sheet.New(&s.tipsOpen, s.tipsBody).Adaptive(true)
	if len(o.Tips) > 0 {
		// Tips paid later (pending, interrupted) are finished as they
		// arrive. An app selling other products sets its own OnUpdate
		// after New, passing purchases to FinishTip.
		store.OnUpdate(func(p store.Purchase) { s.FinishTip(p) })
	}
	return s
}

// Report opens the report form.
func (s *Support) Report() {
	if s.reportOpen {
		return
	}
	s.problem = ""
	s.crash = nil
	if rs := crash.Reports(); len(rs) > 0 {
		r := rs[0]
		s.crash = &r
		s.withCrash = checkbox.New(fmt.Sprintf("Include the crash from %s", ago(r.Time)))
		s.withCrash.SetChecked(true)
	}
	s.withLogs.SetChecked(false)
	s.reportOpen = true
}

// CanDonate reports whether Donate has something to offer here: tips
// where the store requires its own payments, the donation page
// elsewhere. Hide the Donate button when it's false.
func (s *Support) CanDonate() bool {
	if store.BillingRequired() {
		return len(s.o.Tips) > 0
	}
	return s.o.DonateURL != ""
}

// Donate opens the tips sheet where the store requires its own payments,
// and the donation page elsewhere.
func (s *Support) Donate() {
	if store.BillingRequired() {
		if len(s.o.Tips) > 0 {
			s.openTips()
		}
		return
	}
	if s.o.DonateURL == "" {
		return
	}
	go func() {
		if err := s.app.OpenURL(context.Background(), s.o.DonateURL); err != nil {
			s.app.Post(func() { s.toasts.Show(giode.ToastError, "Couldn't open "+s.o.DonateURL) })
		}
	}()
}

// FinishTip finishes p if it's a tip, consuming it so it can be given
// again, and thanks the user; it reports whether p was a tip.
func (s *Support) FinishTip(p store.Purchase) bool {
	if !s.isTip(p.ProductID) {
		return false
	}
	if p.Pending {
		return true
	}
	if err := store.Finish(context.Background(), p, true); err == nil {
		s.app.Post(func() { s.toasts.Show(giode.ToastSuccess, "Thank you for your support!") })
	}
	return true
}

func (s *Support) isTip(id string) bool {
	for _, t := range s.o.Tips {
		if t == id {
			return true
		}
	}
	return false
}

// Layout lays out the open sheets and, unless Options.Toasts was given,
// the toasts.
func (s *Support) Layout(gtx layout.Context) layout.Dimensions {
	s.report.Layout(gtx)
	s.tips.Layout(gtx)
	if s.own {
		s.toasts.Layout(gtx)
	}
	return layout.Dimensions{}
}

// OutOfFlow marks Support as taking no space in a Box.
func (s *Support) OutOfFlow() {}

func (s *Support) reportBody() giode.Element {
	muted := giode.Styles{Color: "theme.muted", FontSize: 13}
	items := []giode.Element{
		giode.Text("Your report is posted on GitHub, where anyone can read it. Your email isn't: it's kept for replies only.", muted),
		s.title,
		s.details,
		s.email,
		s.system,
	}
	if s.system.Checked() {
		items = append(items, giode.Box(giode.Styles{Padding: giode.Inset{Left: 28}}, giode.Text(s.systemLine(), giode.Styles{Color: "theme.muted", FontSize: 12})))
	}
	if s.crash != nil {
		items = append(items, s.withCrash)
	}
	if s.o.Logs != nil {
		items = append(items, s.withLogs)
	}
	if s.problem != "" {
		items = append(items, giode.Text(s.problem, giode.Styles{Color: "theme.danger"}))
	}
	return giode.Stack(giode.Styles{Width: -1, Gap: 12, Padding: giode.Inset{Bottom: 8}}, items...)
}

func (s *Support) systemLine() string {
	var r Report
	r.System()
	parts := []string{}
	if s.o.Name != "" || r.Version != "" {
		parts = append(parts, strings.TrimSpace(s.o.Name+" "+r.Version))
	}
	parts = append(parts, strings.TrimSpace(osName(r.OS)+" "+r.OSVersion), r.Arch)
	if r.Model != "" {
		parts = append(parts, r.Model)
	}
	return strings.Join(parts, " · ")
}

func (s *Support) submit() {
	if s.sending {
		return
	}
	r := Report{
		App:         s.o.App,
		Title:       strings.TrimSpace(s.title.Text()),
		Description: strings.TrimSpace(s.details.Text()),
		Email:       strings.TrimSpace(s.email.Text()),
	}
	switch {
	case r.Title == "":
		s.problem = "Add a short summary."
		s.title.Focus()
		return
	case r.Email != "" && !strings.Contains(r.Email, "@"):
		s.problem = "That email address doesn't look right."
		s.email.Focus()
		return
	case s.o.ReportURL == "":
		s.problem = "Reports aren't set up in this build."
		return
	}
	if s.system.Checked() {
		r.System()
	}
	if s.crash != nil && s.withCrash.Checked() {
		r.Crash = s.crash.Message + "\n\n" + s.crash.Trace
	}
	if s.o.Logs != nil && s.withLogs.Checked() {
		r.Logs = s.o.Logs()
	}
	s.problem = ""
	s.sending = true
	s.send.Loading(true)
	go func() {
		issue, err := Send(context.Background(), s.o.ReportURL, s.o.Key, r)
		s.app.Post(func() {
			s.sending = false
			s.send.Loading(false)
			if err != nil {
				s.problem = failure(err)
				return
			}
			s.reportOpen = false
			s.title.SetText("")
			s.details.SetText("")
			if issue != "" {
				s.toasts.ShowAction(giode.ToastSuccess, "Thanks, your report was sent.", "View", func() {
					go s.app.OpenURL(context.Background(), issue)
				})
			} else {
				s.toasts.Show(giode.ToastSuccess, "Thanks, your report was sent.")
			}
		})
	}()
}

// failure says why a report wasn't sent: the relay's reason in full,
// what answered instead of it, or why it couldn't be reached.
func failure(err error) string {
	var netErr *url.Error
	if errors.As(err, &netErr) {
		return "Couldn't reach the report server (" + netErr.Error() + "). Check your connection and try again."
	}
	msg := "Couldn't send the report: " + strings.TrimPrefix(err.Error(), "support: ")
	if !strings.HasSuffix(msg, ".") {
		msg += "."
	}
	return msg
}

// osName is GOOS the way people write it.
func osName(goos string) string {
	switch goos {
	case "darwin":
		return "macOS"
	case "ios":
		return "iOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "android":
		return "Android"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	}
	return goos
}

// ago describes how long ago t was, roughly.
func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 2*time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 2*time.Hour:
		return "an hour ago"
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	default:
		return t.Format("January 2")
	}
}

var loadMu sync.Mutex

func (s *Support) openTips() {
	s.tipsOpen = true
	if s.loading || len(s.products) > 0 {
		return
	}
	s.loading, s.tipError = true, ""
	go func() {
		loadMu.Lock()
		defer loadMu.Unlock()
		ps, err := store.Products(context.Background(), s.o.Tips...)
		s.app.Post(func() {
			s.loading = false
			if err != nil || len(ps) == 0 {
				s.tipError = "The store's tips couldn't be loaded. Try again later."
				return
			}
			// In the order of Options.Tips.
			byID := map[string]store.Product{}
			for _, p := range ps {
				byID[p.ID] = p
			}
			s.products, s.buttons = nil, nil
			for _, id := range s.o.Tips {
				p, ok := byID[id]
				if !ok {
					continue
				}
				s.products = append(s.products, p)
				var b *giode.ButtonElement
				b = giode.Button(tipLabel(p), giode.Styles{Width: -1}).OnClick(func() { s.buy(p, b) })
				s.buttons = append(s.buttons, b)
			}
		})
	}()
}

func tipLabel(p store.Product) string {
	title := p.Title
	// The App Store and Play add the app's name in parentheses.
	if i := strings.LastIndex(title, " ("); i > 0 && strings.HasSuffix(title, ")") {
		title = title[:i]
	}
	if title == "" {
		return p.Price
	}
	return title + " · " + p.Price
}

func (s *Support) buy(p store.Product, b *giode.ButtonElement) {
	b.Loading(true)
	go func() {
		purchase, err := store.Buy(context.Background(), s.app, p)
		if err == nil && !purchase.Pending {
			err = store.Finish(context.Background(), purchase, true)
		}
		s.app.Post(func() {
			b.Loading(false)
			switch {
			case errors.Is(err, store.ErrCanceled):
			case err != nil:
				s.toasts.Show(giode.ToastError, "The tip didn't go through: "+err.Error())
			case purchase.Pending:
				s.tipsOpen = false
				s.toasts.Show(giode.ToastInfo, "Your tip is waiting for payment. Thank you!")
			default:
				s.tipsOpen = false
				s.toasts.Show(giode.ToastSuccess, "Thank you for your support!")
			}
		})
	}()
}

func (s *Support) tipsBody() giode.Element {
	name := s.o.Name
	if name == "" {
		name = "this app"
	}
	items := []giode.Element{
		giode.H4("Support " + name),
		giode.Text("If "+name+" is useful to you, a tip helps keep it going. It's a one-time payment through the store, and unlocks nothing: the app stays the same for everyone.",
			giode.Styles{Color: "theme.muted"}),
	}
	switch {
	case s.loading:
		items = append(items, giode.Row(giode.Styles{Width: -1, Justify: giode.JustifyCenter, Padding: giode.SymmetricInset(0, 12)}, giode.Spinner()))
	case s.tipError != "":
		items = append(items, giode.Text(s.tipError, giode.Styles{Color: "theme.danger"}))
	default:
		for _, b := range s.buttons {
			items = append(items, b)
		}
	}
	return giode.Stack(giode.Styles{Width: -1, Gap: 12, Padding: giode.Inset{Bottom: 8}}, items...)
}
