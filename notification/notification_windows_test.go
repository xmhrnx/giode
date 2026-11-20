package notification

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func TestToastXML(t *testing.T) {
	x := toastXML(Notification{ID: "m1", Title: "A & B", Body: `"hi" <3`, Silent: true, Data: map[string]string{"k": "v"},
		Actions: []Action{{ID: "like", Title: "Like"}, {ID: "reply", Title: "Reply", Reply: true, Placeholder: "Message"}, {ID: "open", Title: "Open", Foreground: true}}})
	for _, want := range []string{
		`<text>A &amp; B</text>`, `<text>&#34;hi&#34; &lt;3</text>`,
		`<input id="r" type="text" placeHolderContent="Message"/>`,
		`hint-inputId="r"`, `activationType="foreground"/>`, `<audio silent="true"/>`,
		`arguments="giode.n:{&#34;i&#34;:&#34;m1&#34;,&#34;d&#34;:{&#34;k&#34;:&#34;v&#34;},&#34;a&#34;:&#34;like&#34;}"`,
	} {
		if !strings.Contains(x, want) {
			t.Errorf("toast XML lacks %s:\n%s", want, x)
		}
	}
	if strings.Count(x, "<input") != 1 {
		t.Error("one reply box expected")
	}
}

func TestActivated(t *testing.T) {
	got := make(chan Response, 1)
	mu.Lock()
	handler = func(r Response) { got <- r }
	mu.Unlock()
	t.Cleanup(func() { mu.Lock(); handler = nil; mu.Unlock() })
	if activated("other", nil) {
		t.Error("took someone else's activation")
	}
	if !activated(`giode.n:{"i":"m1","d":{"k":"v"},"a":"reply"}`, map[string]string{"r": "on my way"}) {
		t.Fatal("didn't take its activation")
	}
	select {
	case r := <-got:
		if r.ID != "m1" || r.Action != "reply" || r.Reply != "on my way" || r.Data["k"] != "v" {
			t.Errorf("%+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no response")
	}
}

// TestShowToast shows, replaces and removes a real toast. Hosted runners
// may refuse toasts; the calls must still return.
func TestShowToast(t *testing.T) {
	aumid, err := setup()
	if err != nil {
		t.Skip("toast setup:", err)
	}
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\`+aumid)
	})
	id, err := Show(Notification{Title: "giode test", Body: "toast", Thread: "t", Actions: []Action{{ID: "a", Title: "A"}}})
	if err != nil {
		t.Log("show:", err)
		return
	}
	if _, err := Show(Notification{ID: id, Title: "giode test", Body: "replaced", Thread: "t"}); err != nil {
		t.Error("replace:", err)
	}
	if err := Remove(id); err != nil {
		t.Error("remove:", err)
	}
	if err := RemoveAll(); err != nil {
		t.Error("remove all:", err)
	}
	if err := SetBadge(3); err != nil {
		t.Log("badge:", err)
	}
}
