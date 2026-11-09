# Reports and donations

The `support` package gives an app a **Report a problem** form and a **Donate** button.

- Reports become GitHub issues in the app's repository. They go through a small relay you
  deploy once for all your apps, so users don't need a GitHub account and the GitHub
  token never ships inside the app.
- Donate opens your donation page (Ko-fi, Buy Me a Coffee, GitHub Sponsors). In stores
  that require their own payments, it offers tips as in-app purchases instead.

```go
import "github.com/xmhrnx/giode/support"

help := support.New(app, support.Options{
	ReportURL: "https://giode-support.you.workers.dev/report",
	App:       "sonora",                 // the relay's name for this app
	DonateURL: "https://ko-fi.com/you",
	Tips:      []string{"tip_coffee", "tip_lunch", "tip_dinner"},
})

report := giode.Button("Report a problem").OnClick(help.Report)
donate := giode.Button("Donate").OnClick(help.Donate)

app.Run(func() giode.Element {
	items := []giode.Element{page, report}
	if help.CanDonate() {
		items = append(items, donate)
	}
	// help takes no space: it paints its forms and toasts over the window.
	return giode.Stack(giode.Styles{}, append(items, help)...)
})
```

## The report form

`Report()` opens a sheet, shown as a centered dialog on wide windows. It asks for:

- a summary;
- details;
- an email address for replies, which is optional and never published;
- whether to include the app version and system details (on by default; the form shows
  exactly what will be sent);
- the latest crash, when [package crash](platform.md#crash-reports) recorded one;
- recent logs, when `Options.Logs` is set.

While a report sends, the Send button spins. When it's sent, a toast thanks the user and
links to the issue. If sending fails, the form stays open with its text and says why: the
connection, or the relay's reason (too many reports, say).

To send a report without the form, use
`support.Send(ctx, url, key, support.Report{...})`. `Report.System()` fills in the
version, build, OS, CPU, device model and locale.

## The relay

The relay is a Cloudflare Worker in [`support/relay`](../support/relay/README.md),
deployed once for all your apps. Cloudflare's free plan covers it. It:

- opens an issue in the app's repository, labelled `user report` (plus `crash` when a
  crash is attached);
- rate-limits each network, by default to 5 reports an hour and 20 a day;
- neutralises `@mentions` and `#123` links in the user's text, so a report can't ping
  anyone;
- keeps the reporter's email out of the public issue. It's stored in the Worker for a
  year, and you look it up with your admin token.

Its README has the five-minute setup: a fine-grained GitHub token with **Issues: Read and
write**, a KV namespace, the `APPS` list of app names and repositories, and
`npx wrangler deploy`.

## Donations and store rules

The App Store, the Mac App Store and Google Play don't allow linking to an outside page to
pay for tips: tips must be in-app purchases there. The Microsoft Store and direct
downloads allow donation links. `Donate()` follows these rules on its own.

| Installed from | Donate opens |
| --- | --- |
| App Store (iOS), Mac App Store, Google Play | A sheet of `Options.Tips`, with the store's titles and local prices |
| Microsoft Store, a direct download, Linux packages, the web | `Options.DonateURL` |

`store.BillingRequired()` reports which case applies. It's true for every iOS app, for a
Mac app with an App Store receipt (Developer ID apps have none), and for an Android app
that Google Play installed. When there's nothing to offer, `CanDonate()` is false: tips
needed but none set, or no `DonateURL`. Hide the button then.

To set up the tips:

1. Create **consumable** in-app products: in App Store Connect under the app's In-App
   Purchases, and in the Play Console under Monetize → In-app products. Give them the
   same ids everywhere, such as `tip_coffee`, `tip_lunch` and `tip_dinner`, with prices
   and titles ("Coffee", "Lunch").
2. List the ids in `Options.Tips`, in the order to show them.

A tip is consumed as soon as it's paid, so it can be given again, and it unlocks nothing.
Apple and Google both require that: a tip must not buy features. Tips paid later (pending
payments, interrupted purchases) are finished through `store.OnUpdate`, which `New` sets.
An app that also sells other products sets its own `store.OnUpdate` after `New`, and
passes each purchase to `help.FinishTip(p)` first: it returns true for tips.

On Apple's platforms, the app's ID needs the In-App Purchase capability in your developer
account, which is on by default. On Android, the store package brings in Google Play
Billing and its permission.
