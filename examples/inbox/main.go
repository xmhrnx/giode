// Command inbox shows rich notifications: a message with Like and
// Reply buttons, threads that group by conversation, a quiet channel,
// and the badge count. What the user does with them, also with the app
// closed, shows in the log. A background task posts a digest every
// 15 minutes, also while the app isn't running (giode.toml lists it in
// app.background_tasks for iOS).
package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/background"
	"github.com/xmhrnx/giode/notification"
)

func main() {
	var (
		app    *giode.App
		lines  []string
		unread int
		next   int
	)
	logf := func(format string, args ...any) {
		lines = append([]string{time.Now().Format("15:04:05") + "  " + fmt.Sprintf(format, args...)}, lines...)
		if len(lines) > 12 {
			lines = lines[:12]
		}
	}
	people := []string{"Ana", "Bo", "Cy"}
	message := func(from string) {
		next++
		unread++
		id, err := notification.Show(notification.Notification{
			Title:   from,
			Body:    fmt.Sprintf("Message %d: lunch at one?", next),
			Thread:  "chat-" + strings.ToLower(from),
			Channel: "messages",
			Badge:   unread,
			Data:    map[string]string{"from": from},
			Actions: []notification.Action{
				{ID: "like", Title: "Like"},
				{ID: "reply", Title: "Reply", Reply: true, Placeholder: "Message " + from},
			},
		})
		if err != nil {
			logf("show: %v", err)
			return
		}
		logf("sent %s from %s", id, from)
	}

	// The system may start the process just for this: register it
	// before Run.
	background.Register("digest", func(ctx context.Context) error {
		_, err := notification.Show(notification.Notification{ID: "digest", Title: "Inbox", Channel: "digest",
			Body: "Checked in the background at " + time.Now().Format("15:04")})
		return err
	})

	app = giode.New(giode.Config{Title: "Inbox", Size: image.Pt(460, 620), Background: "theme.background"})
	notification.DefineChannel(notification.Channel{ID: "messages", Name: "Messages", Description: "New messages", Importance: notification.High})
	notification.DefineChannel(notification.Channel{ID: "digest", Name: "Digest", Description: "Daily summaries", Importance: notification.Low})
	notification.Listen(app, func(r notification.Response) {
		app.Post(func() {
			switch {
			case r.Action == "reply":
				logf("%s: reply to %s: %q", r.ID, r.Data["from"], r.Reply)
				// Showing it again keeps it up, with the reply.
				notification.Show(notification.Notification{ID: r.ID, Title: r.Data["from"], Body: "You: " + r.Reply,
					Thread: "chat-" + strings.ToLower(r.Data["from"]), Channel: "messages", Silent: true, Data: r.Data})
			case r.Action != "":
				logf("%s: %s", r.ID, r.Action)
			default:
				logf("%s: opened", r.ID)
				unread = 0
				notification.SetBadge(0)
			}
		})
	})

	ask := giode.Button("Allow notifications").OnClick(func() {
		go func() {
			ok, err := app.RequestNotificationPermission(context.Background())
			app.Post(func() { logf("allowed: %v %v", ok, err) })
		}()
	})
	send := giode.Button("Message in 3 s").OnClick(func() {
		from := people[next%len(people)]
		logf("switch away: %s writes in 3 seconds", from)
		go func() {
			time.Sleep(3 * time.Second)
			app.Post(func() { message(from) })
		}()
	})
	digest := giode.Button("Quiet digest").OnClick(func() {
		if _, err := notification.Show(notification.Notification{ID: "digest", Title: "Today", Body: "3 conversations, 7 messages", Channel: "digest"}); err != nil {
			logf("digest: %v", err)
		}
	})
	every := giode.Button("Digest every 15 min").OnClick(func() {
		err := background.Schedule("digest", background.Options{Interval: 15 * time.Minute, Network: true})
		logf("background digest: %v", err)
	})
	clear := giode.Button("Clear all").OnClick(func() {
		unread = 0
		notification.SetBadge(0)
		if err := notification.RemoveAll(); err != nil {
			logf("clear: %v", err)
		}
	})

	err := app.Run(func() giode.Element {
		items := []giode.Element{
			giode.Row(giode.Styles{Gap: 8, Wrap: giode.WrapWrap}, ask, send, digest, every, clear),
			giode.Text(fmt.Sprintf("Unread: %d", unread), giode.Styles{Color: "theme.muted"}),
		}
		for _, l := range lines {
			items = append(items, giode.Text(l, giode.Styles{FontSize: 13}))
		}
		return giode.Stack(giode.Styles{Width: -1, Height: -1, Padding: giode.UniformInset(20), Gap: 10}, items...)
	})
	if err != nil {
		log.Fatal(err)
	}
}
