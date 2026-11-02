package giode

import "testing"

func TestTraySetters(t *testing.T) {
	a := New(Config{Title: "t", CloseToTray: true})
	if !a.CloseToTray() || a.MinimizeToTray() {
		t.Fatal("setters don't start from Config")
	}
	a.SetCloseToTray(false)
	a.SetMinimizeToTray(true)
	if a.CloseToTray() || !a.MinimizeToTray() {
		t.Fatal("setters didn't apply")
	}
	if a.closesToTray() {
		t.Error("closesToTray after SetCloseToTray(false)")
	}
	if a.minimizesToTray() != traySupported {
		t.Error("minimizesToTray doesn't follow SetMinimizeToTray")
	}
}

func TestStartInTray(t *testing.T) {
	defer launchedAtLogin.Store(launchedAtLogin.Load())
	launchedAtLogin.Store(false)
	if StartInTrayNever.startsInTray() || StartInTrayAtLogin.startsInTray() {
		t.Error("starts in tray without a login launch")
	}
	if StartInTrayAlways.startsInTray() != traySupported {
		t.Error("StartInTrayAlways")
	}
	launchedAtLogin.Store(true)
	if !LaunchedAtLogin() || StartInTrayAtLogin.startsInTray() != traySupported {
		t.Error("StartInTrayAtLogin at login")
	}
}

func TestLoginName(t *testing.T) {
	defer func(id string) { AppID = id }(AppID)
	AppID = ""
	a := New(Config{Title: "My App: 2/3"})
	if got := a.loginName(); got != "My App- 2-3" {
		t.Errorf("loginName from title = %q", got)
	}
	AppID = "com.example.app"
	if got := a.loginName(); got != "com.example.app" {
		t.Errorf("loginName from AppID = %q", got)
	}
}

func TestStripLoginArg(t *testing.T) {
	args, ok := stripLoginArg([]string{"app", "--giode-login", "file.txt"})
	if !ok || len(args) != 2 || args[1] != "file.txt" {
		t.Errorf("got %q %v", args, ok)
	}
	if _, ok := stripLoginArg([]string{"--giode-login"}); ok {
		t.Error("the program name taken as the login argument")
	}
}

func TestTrayClickShowsHiddenWindowDirectly(t *testing.T) {
	a := New(Config{Title: "t"})
	a.windowHidden.Store(true)
	a.Tray().clicked()
	if a.windowHidden.Load() {
		t.Error("the default tray click waited for the UI goroutine to show the window")
	}
}

func TestSecondLaunchShowsHiddenWindowDirectly(t *testing.T) {
	a := New(Config{Title: "t"})
	a.windowHidden.Store(true)
	a.handleLaunch([]string{"file.txt"})
	if a.windowHidden.Load() {
		t.Error("a second launch didn't show the hidden window straight away")
	}
}

func TestHideWindowBeforeWindow(t *testing.T) {
	a := New(Config{Title: "t"})
	a.HideWindow()
	a.viewMu.Lock()
	pending := a.hidePending
	a.viewMu.Unlock()
	if !pending || a.WindowVisible() {
		t.Error("HideWindow before the window exists isn't kept for when it does")
	}
	a.ShowWindow()
	a.viewMu.Lock()
	pending = a.hidePending
	a.viewMu.Unlock()
	if pending || !a.WindowVisible() {
		t.Error("ShowWindow didn't cancel the pending hide")
	}
}

func TestRunUICallsReportsWork(t *testing.T) {
	a := New(Config{Title: "t"})
	if a.runUICalls() {
		t.Error("ran calls with none queued")
	}
	ran := false
	a.Post(func() { ran = true })
	if !a.runUICalls() || !ran {
		t.Error("queued call didn't run")
	}
}
