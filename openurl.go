package giode

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/xmhrnx/giode/internal/platform/shell"
)

// allowedSchemes are the URL schemes OpenURL opens. Links in content the
// app doesn't control (a model's reply, a web page) must not reach
// file:, javascript: or an arbitrary app's handler.
var allowedSchemes = map[string]bool{"http": true, "https": true, "mailto": true, "tel": true, "sms": true}

// OpenURL opens rawURL in the app that handles it - the browser for web
// links, the mail or phone app for mailto: and tel: - on every
// platform: an ACTION_VIEW intent on Android, UIApplication openURL on
// iOS, and open, xdg-open or the URL handler on desktops. Only http,
// https, mailto, tel and sms links are opened; others return an error,
// so links from untrusted content can't reach local files or scripts.
//
// Links in RichText and Markdown open through it unless the paragraph
// has its own OnLink handler.
func (a *App) OpenURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("giode: invalid URL %q: %w", rawURL, err)
	}
	if !allowedSchemes[strings.ToLower(u.Scheme)] {
		return fmt.Errorf("giode: won't open %q: only http, https, mailto, tel and sms links are opened", rawURL)
	}
	return shell.OpenURL(ctx, a.host(), u.String())
}

// openLink is the default handler for RichText links.
func (a *App) openLink(u string) {
	go func() {
		if err := a.OpenURL(context.Background(), u); err != nil {
			a.logf(LogLevelWarning, "giode: opening link: %v", err)
		}
	}()
}
