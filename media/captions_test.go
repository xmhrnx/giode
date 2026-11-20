package media

import (
	"strings"
	"testing"
	"time"
)

func TestParseSRT(t *testing.T) {
	src := "\uFEFF1\r\n00:00:01,000 --> 00:00:02,500\r\n<i>Hello</i>\r\nthere\r\n\r\n2\r\n00:00:03,000 --> 00:00:04,000\r\n{\\an8}Top &amp; bottom\r\n"
	cues, err := ParseCaptions(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []Cue{
		{time.Second, 2500 * time.Millisecond, "Hello\nthere"},
		{3 * time.Second, 4 * time.Second, "Top & bottom"},
	}
	if len(cues) != len(want) {
		t.Fatalf("cues = %+v", cues)
	}
	for i := range want {
		if cues[i] != want[i] {
			t.Errorf("cue %d = %+v, want %+v", i, cues[i], want[i])
		}
	}
}

func TestParseVTT(t *testing.T) {
	src := `WEBVTT - a title
Kind: captions

NOTE this is skipped
and so is this

intro
00:01.500 --> 00:03.000 align:start position:10%
<v Ada>Hi</v> <c.loud>all</c>

01:00:00.000 --> 01:00:01.250
Late
`
	cues, err := ParseCaptions(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0] != (Cue{1500 * time.Millisecond, 3 * time.Second, "Hi all"}) ||
		cues[1] != (Cue{time.Hour, time.Hour + 1250*time.Millisecond, "Late"}) {
		t.Errorf("cues = %+v", cues)
	}
	if _, err := ParseCaptions(strings.NewReader("WEBVTT\n\n")); err == nil {
		t.Error("an empty file parsed")
	}
}

func TestCueAt(t *testing.T) {
	cues := []Cue{{0, 2 * time.Second, "a"}, {time.Second, 3 * time.Second, "b"}, {5 * time.Second, 6 * time.Second, "c"}}
	for _, tc := range []struct {
		at   time.Duration
		want string
	}{{500 * time.Millisecond, "a"}, {1500 * time.Millisecond, "a\nb"}, {4 * time.Second, ""}, {5 * time.Second, "c"}, {6 * time.Second, ""}} {
		if got := cueAt(cues, tc.at); got != tc.want {
			t.Errorf("at %v: %q, want %q", tc.at, got, tc.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	for d, want := range map[time.Duration]string{0: "0:00", 65 * time.Second: "1:05", 3725 * time.Second: "1:02:05", -time.Second: "0:00"} {
		if got := FormatTime(d); got != want {
			t.Errorf("FormatTime(%v) = %q", d, got)
		}
	}
	if speedLabel(1.25) != "1.25×" || speedLabel(2) != "2×" || speedLabel(0.5) != "0.5×" {
		t.Error(speedLabel(1.25), speedLabel(2), speedLabel(0.5))
	}
}
