package giode

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/xmhrnx/giode/auth"
)

func TestAuthRedirectSkipsOnOpenURL(t *testing.T) {
	opened := make(chan *url.URL, 1)
	a := &App{cfg: Config{OnOpenURL: func(u *url.URL) { opened <- u }}}
	host := appAuthHost{a}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got := make(chan *url.URL, 1)
	go func() {
		u, err := host.WaitForURL(ctx, func(u *url.URL) bool { return u.Scheme == "com.example.app" })
		if err != nil {
			t.Error(err)
		}
		got <- u
	}()
	for !a.auth.waiting() {
		time.Sleep(time.Millisecond)
	}
	u, _ := url.Parse("com.example.app:/oauth?code=1&state=s")
	if !a.auth.take(u) {
		t.Fatal("the redirect wasn't taken")
	}
	if r := <-got; r.String() != u.String() {
		t.Errorf("got %v", r)
	}
	// Once taken, links go to the app again.
	if a.auth.take(u) || a.auth.waiting() {
		t.Error("a second link was taken")
	}
	// A canceled wait leaves nothing behind.
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if _, err := host.WaitForURL(ctx2, func(*url.URL) bool { return true }); err == nil || a.auth.waiting() {
		t.Errorf("canceled wait: %v, waiting %v", err, a.auth.waiting())
	}
}

func TestAuthSessionsPerName(t *testing.T) {
	a := &App{}
	g := a.Auth(auth.Google("1234.apps.googleusercontent.com"))
	if a.Auth(auth.Google("1234.apps.googleusercontent.com")) != g {
		t.Error("a second session for the same provider")
	}
	if a.Auth(auth.Microsoft("", "id")) == g {
		t.Error("providers share a session")
	}
}
