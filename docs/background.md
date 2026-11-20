# Background work

The `background` package runs the app's work while it isn't on screen: a sync every hour or
so, a cleanup at night while charging. The system decides when, within the constraints the
app gives, and starts the app's process for it when it isn't running.

```go
import "github.com/xmhrnx/giode/background"

func main() {
	// The system may start the process just to run a task: register
	// it before App.Run.
	background.Register("sync", func(ctx context.Context) error {
		return syncMail(ctx) // return when ctx is canceled: the time is up
	})

	app := giode.New(giode.Config{Title: "Mail"})
	background.Schedule("sync", background.Options{
		Interval: time.Hour, // again about every hour; zero runs it once
		Network:  true,      // only with a connection
	})
	app.Run(view)
}
```

`Options` also has `Delay` (the earliest first run), `Charging` (only while charging) and
`Long` (minutes rather than seconds). `Cancel` removes a schedule. An error from the task
asks Android to try again later; on iOS it reports a failed run. To tell the user something,
show a [notification](notifications.md) from the task.

| | How | Time per run | Notes |
|---|---|---|---|
| iOS | BGTaskScheduler: an app refresh, or with `Long` a processing task | about 30 s; minutes for `Long`, usually at night while charging | The system learns when the app is used and picks its moments; `Interval` is the earliest next run. |
| Android | JobScheduler | up to about 10 minutes | Periodic jobs run at most every 15 minutes, and survive reboots. |
| Computers | a timer while the app runs | no limit | Apps there keep running in the tray or in the background on their own (see `Config.CloseToTray`). |

## Setup

iOS permits only the tasks the app declares. List their names in `giode.toml`:

```toml
[app]
background_tasks = ["sync"]
```

`giode build` adds them as `BGTaskSchedulerPermittedIdentifiers` (`<app.id>.sync`) with the
`fetch` and `processing` background modes. Android needs nothing: the package's manifest
declares its job service.

To try a task on iOS without waiting for the system, pause the app in Xcode's debugger and
run `e -l objc -- (void)[[BGTaskScheduler sharedScheduler] _simulateLaunchForTaskWithIdentifier:@"<app.id>.sync"]`.
On Android, `adb shell cmd jobscheduler run -f <app.id> <job id>` runs a job now; the job
id is the Java hash of `giode.background.<name>`.

See `examples/inbox`.
