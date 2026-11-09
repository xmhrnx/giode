// Package support lets the app's users report a problem and support
// its development.
//
// Report opens a form whose reports become GitHub issues through a small
// relay you deploy (see support/relay: a Cloudflare Worker), so users
// need no GitHub account and the token never ships in the app. Donate
// opens your donation page (Ko-fi, say), or, where the store requires
// its own payments (the App Store, the Mac App Store, Google Play), a
// sheet of tips sold as consumable in-app purchases.
//
//	help := support.New(app, support.Options{
//		ReportURL: "https://support.example.workers.dev/report",
//		App:       "sonora",
//		DonateURL: "https://ko-fi.com/you",
//		Tips:      []string{"tip_small", "tip_medium", "tip_large"},
//	})
//	reportButton := giode.Button("Report a problem").OnClick(help.Report)
//	donateButton := giode.Button("Donate").OnClick(help.Donate)
//
//	// In the view: the button where it belongs, and help anywhere (it
//	// takes no space, and paints its sheets over the window).
//	giode.Stack(giode.Styles{}, page, help)
package support

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/xmhrnx/giode"
)

// Options configures the report form and Donate.
type Options struct {
	// ReportURL is the relay's report address, ending in /report.
	ReportURL string
	// App names the app to the relay, which maps it to a repository:
	// one relay serves all your apps. Default giode.AppID.
	App string
	// Key, when set, is sent with each report for the relay's KEY to
	// check: it keeps out reports not sent by your apps. It ships in the
	// app, so it stops casual spam, not a determined sender.
	Key string
	// Name is the app's name in the forms; default giode.AppName.
	Name string
	// DonateURL is the page Donate opens: "https://ko-fi.com/you".
	DonateURL string
	// Tips are the ids of consumable in-app products Donate offers
	// instead of DonateURL where the store requires its own payments
	// (store.BillingRequired). Create them in App Store Connect and the
	// Play Console; they're shown in this order, with the store's
	// titles and prices.
	Tips []string
	// Logs, when set, returns recent log lines the form offers to
	// attach.
	Logs func() string
	// Toasts shows the forms' confirmations and errors; by default
	// Support shows them in a toaster of its own.
	Toasts *giode.ToasterElement
}

// Report is a problem report, as the relay receives it.
type Report struct {
	App         string `json:"app"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Email is for replies: the relay keeps it out of the issue.
	Email string `json:"email,omitempty"`
	// The app and system, when the user agrees to include them.
	Version   string `json:"version,omitempty"`
	Build     string `json:"build,omitempty"`
	OS        string `json:"os,omitempty"`
	OSVersion string `json:"os_version,omitempty"`
	Arch      string `json:"arch,omitempty"`
	Model     string `json:"model,omitempty"`
	Locale    string `json:"locale,omitempty"`
	// Crash is the text of a crash report (see package crash), Logs
	// recent log lines.
	Crash string `json:"crash,omitempty"`
	Logs  string `json:"logs,omitempty"`
}

// Limits the relay enforces too.
const (
	maxTitle       = 120
	maxDescription = 20000
	maxAttachment  = 30000
)

// System fills in the app and system fields: the app's version and
// build, the OS, its version, the CPU, the device model and the
// locale.
func (r *Report) System() {
	d := giode.Device()
	r.Version, r.Build = giode.AppVersion, giode.AppBuild
	r.OS, r.OSVersion, r.Arch, r.Model = d.OS, d.OSVersion, runtime.GOARCH, d.Model
	r.Locale = giode.Locale()
}

// ErrInvalid is returned for a report the relay would turn away: no
// app or title, or a description too long.
var ErrInvalid = errors.New("support: invalid report")

// Send posts r to the relay at url and returns the issue's address.
func Send(ctx context.Context, url, key string, r Report) (string, error) {
	r.Title = strings.TrimSpace(r.Title)
	if r.App == "" || r.Title == "" || len(r.Description) > maxDescription {
		return "", ErrInvalid
	}
	if len(r.Title) > maxTitle {
		cut := maxTitle
		for cut > 0 && r.Title[cut]&0xC0 == 0x80 {
			cut--
		}
		r.Title = r.Title[:cut]
	}
	r.Crash = truncate(r.Crash, maxAttachment)
	r.Logs = truncate(r.Logs, maxAttachment)
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("X-Support-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var answer struct {
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&answer)
	if resp.StatusCode/100 != 2 {
		if answer.Error != "" {
			return "", fmt.Errorf("support: %s", answer.Error)
		}
		return "", fmt.Errorf("support: sending the report: %s", resp.Status)
	}
	return answer.URL, nil
}

// truncate shortens s to at most n bytes, at a rune boundary, keeping
// its end for long text (where a trace's crash is).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := len(s) - n + len("…\n")
	for cut < len(s) && s[cut]&0xC0 == 0x80 {
		cut++
	}
	return "…\n" + s[cut:]
}
