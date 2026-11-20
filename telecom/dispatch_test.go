package telecom

import (
	"regexp"
	"testing"
)

func TestDispatch(t *testing.T) {
	var got []string
	state.h = Handler{
		OnAnswer: func(id string) { got = append(got, "answer "+id) },
		OnEnd:    func(id string) { got = append(got, "end "+id) },
		OnMute: func(id string, m bool) {
			if m {
				got = append(got, "mute "+id)
			} else {
				got = append(got, "unmute "+id)
			}
		},
		OnStart: func(c Call) { got = append(got, "start "+c.ID+" "+c.Handle) },
		OnAudio: func(on bool) {
			if on {
				got = append(got, "audio")
			}
		},
	}
	defer func() { state.h = Handler{} }()
	// Without an app, deliver runs on a goroutine: run it here instead.
	for _, e := range [][3]string{{"answer", "A", ""}, {"mute", "A", ""}, {"unmute", "A", ""}, {"start", "B", "ada"}, {"audio_on", "", ""}, {"end", "A", ""}, {"unknown", "", ""}} {
		dispatchSync(e[0], e[1], e[2])
	}
	want := []string{"answer A", "mute A", "unmute A", "start B ada", "audio", "end A"}
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d: %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewIDAndPayload(t *testing.T) {
	id := NewID()
	if !regexp.MustCompile(`^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`).MatchString(id) {
		t.Errorf("id %q", id)
	}
	c := callFromPayload(map[string]any{"handle": "ada", "video": true})
	if c.ID == "" || c.Name != "ada" || !c.Video {
		t.Errorf("call %+v", c)
	}
	state.h.OnVoIPPush = func(p map[string]any) Call { return Call{Handle: p["from"].(string)} }
	defer func() { state.h = Handler{} }()
	if c := callFromPayload(map[string]any{"from": "bob"}); c.Handle != "bob" || c.Name != "bob" || c.ID == "" {
		t.Errorf("custom call %+v", c)
	}
}

func TestDesktopUnsupported(t *testing.T) {
	if supported {
		t.Skip("a phone")
	}
	if Supported() || Setup(nil, Handler{}) != ErrNotSupported || ReportIncoming(nil, Call{}) != ErrNotSupported {
		t.Error("desktop should report no system call screen")
	}
}
