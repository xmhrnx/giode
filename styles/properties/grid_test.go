package properties

import "testing"

func TestGridTracks(t *testing.T) {
	for _, c := range []struct {
		spec string
		n    int
		fill bool
		bad  bool
	}{
		{"200 1fr 2fr", 3, false, false},
		{"auto 1fr", 2, false, false},
		{"repeat(3, 1fr)", 3, false, false},
		{"repeat(2, 100 1fr) auto", 5, false, false},
		{"minmax(160, 1fr) 2fr", 2, false, false},
		{"repeat(auto-fill, minmax(220, 1fr))", 1, true, false},
		{"200px 3dp", 2, false, false},
		{"1fr wide", 0, false, true},
		{"repeat(3, 1fr", 0, false, true},
		{"repeat(auto-fill, 1fr) 2fr", 0, false, true},
		{"minmax(1fr, 2fr)", 0, false, true},
	} {
		err := GridTracks(c.spec).Check()
		if c.bad {
			if err == nil {
				t.Errorf("%q: no error", c.spec)
			}
			continue
		}
		p := GridTracks(c.spec).Parse()
		if err != nil || len(p.Tracks) != c.n || p.AutoFill != c.fill {
			t.Errorf("%q: %+v, %v", c.spec, p, err)
		}
	}
	p := GridTracks("minmax(160, 1fr) 200").Parse()
	if p.Tracks[0] != (Track{Kind: TrackFr, Size: 1, Min: 160}) || p.Tracks[1] != (Track{Kind: TrackFixed, Size: 200}) {
		t.Errorf("tracks %+v", p.Tracks)
	}
}
