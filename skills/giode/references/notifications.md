# Notifications

`App.Notify` shows a plain notification: a title and a body. The `notification` package
shows richer ones, with buttons, a reply field, threads, channels, replacement and the
badge count. It also tells the app what the user did with them, including when that
started the app.

```go
import "github.com/xmhrnx/giode/notification"

func main() {
	app := giode.New(giode.Config{Title: "Chat"})
	notification.DefineChannel(notification.Channel{ID: "messages", Name: "Messages", Importance: notification.High})
	notification.Listen(app, func(r notification.Response) {
		app.Post(func() {
			switch r.Action {
			case "reply":
				send(r.Data["chat"], r.Reply) // typed in the notification
			case "read":
				markRead(r.Data["chat"])
			case "": // the notification itself was tapped
				openChat(r.Data["chat"])
			}
		})
	})
	app.Run(view)
}

notification.Show(notification.Notification{
	ID:      "chat-42",               // showing the same ID again replaces it
	Title:   "Ana",
	Body:    "Lunch at one?",
	Thread:  "chat-42",               // groups a conversation
	Channel: "messages",
	Badge:   unread,                  // the app icon's count
	Data:    map[string]string{"chat": "42"},
	Actions: []notification.Action{
		{ID: "read", Title: "Mark as read"},
		{ID: "reply", Title: "Reply", Reply: true, Placeholder: "Message Ana"},
	},
})
notification.Remove("chat-42")
notification.SetBadge(0)
```

Call `Listen` early, before `App.Run`: a response that started the app is delivered once
it's called, and on Windows `Listen` registers the class that Windows calls with button
presses. The handler runs on its own goroutine. Ask for permission with
`App.RequestNotificationPermission` (Android 13 and later, iOS, macOS); without it `Show`
returns `giode.ErrNotificationsOff`.

## What each system does

| | Android | iOS, macOS | Windows | Linux |
|---|---|---|---|---|
| Buttons | up to 3 | yes (a category per set of buttons) | up to 5 | where the service has actions |
| Reply field | Android 7+ | yes | yes | KDE Plasma (others: a plain button) |
| Threads | grouped with a summary | thread identifier | toast group | no |
| Channels | notification channels | interruption level | Low: no popup | urgency and category |
| Badge | the notification's number | app icon, Dock tile | number on the taskbar button | launcher entry (KDE, Ubuntu dock) |
| Buttons with the app closed | starts the process, not the window | launches in the background | starts the app | no |

A button is handled without showing the app unless the action is `Foreground`. On
Android the handler gets about eight seconds; if it shows the notification again (with
the same ID, say with the reply added) it stays, otherwise it's dismissed. On iOS the
system gives a background launch about 30 seconds.

Android's channels are settings the user owns: once a channel exists, its importance can't
be changed from the app, only its name and description. A `Silent` notification on
Android 8 and later goes through a quiet "giode.silent" channel, since sounds belong to
channels there.

On Windows, an app run outside an MSIX package registers its AppUserModelID and its toast
activator under `HKEY_CURRENT_USER` the first time; `giode build` declares them in the
MSIX manifest. The ID is the app's id from `giode.toml` (or the executable's name).

## Buttons on pushes

A push can carry buttons too. On iOS and macOS, register them under a category and name
it in the APNs payload's `"category"`:

```go
notification.DefineCategory("message", notification.Action{ID: "reply", Title: "Reply", Reply: true})
```

The response comes to `Listen`, with the push's custom string keys as `Data`. On Android,
send a data message and show it with `notification.Show` from `push.Handlers.Message`.

See `examples/inbox`.
