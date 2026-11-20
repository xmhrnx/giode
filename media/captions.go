package media

import (
	"bufio"
	"errors"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Cue is one caption: Text shows from Start to End.
type Cue struct {
	Start, End time.Duration
	Text       string
}

// Track is a set of captions in one language.
type Track struct {
	Label    string
	Language string
	Cues     []Cue
}

// ParseCaptions reads SRT or WebVTT captions.
func ParseCaptions(r io.Reader) ([]Cue, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	var (
		cues  []Cue
		cur   *Cue
		lines []string
		first = true
	)
	flush := func() {
		if cur != nil {
			cur.Text = strings.Join(lines, "\n")
			if cur.Text != "" {
				cues = append(cues, *cur)
			}
		}
		cur, lines = nil, nil
	}
	skipBlock := false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if first {
			line = strings.TrimPrefix(line, "\uFEFF")
			first = false
			if strings.HasPrefix(line, "WEBVTT") {
				skipBlock = true // the header block
				continue
			}
		}
		if strings.TrimSpace(line) == "" {
			flush()
			skipBlock = false
			continue
		}
		if skipBlock {
			continue
		}
		if cur == nil {
			if strings.HasPrefix(line, "NOTE") || strings.HasPrefix(line, "STYLE") || strings.HasPrefix(line, "REGION") {
				skipBlock = true
				continue
			}
			if start, end, ok := parseTiming(line); ok {
				cur = &Cue{Start: start, End: end}
			}
			// Otherwise a cue number or identifier.
			continue
		}
		lines = append(lines, stripTags(line))
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(cues) == 0 {
		return nil, errors.New("media: no captions found")
	}
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].Start < cues[j].Start })
	return cues, nil
}

func parseTiming(line string) (start, end time.Duration, ok bool) {
	a, b, found := strings.Cut(line, "-->")
	if !found {
		return 0, 0, false
	}
	start, ok1 := parseStamp(strings.TrimSpace(a))
	// WebVTT puts cue settings after the end time.
	f := strings.Fields(b)
	if len(f) == 0 {
		return 0, 0, false
	}
	end, ok2 := parseStamp(f[0])
	return start, end, ok1 && ok2 && end >= start
}

// parseStamp reads hh:mm:ss,mmm (SRT) or [hh:]mm:ss.mmm (WebVTT).
func parseStamp(s string) (time.Duration, bool) {
	s = strings.Replace(s, ",", ".", 1)
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	units := []time.Duration{time.Hour, time.Minute}[3-len(parts):]
	var d time.Duration
	for i, p := range parts[:len(parts)-1] {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, false
		}
		d += time.Duration(n) * units[i]
	}
	sec, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil {
		return 0, false
	}
	return d + time.Duration(math.Round(sec*1000))*time.Millisecond, true
}

var tagRE = regexp.MustCompile(`<[^>]*>|\{\\[^}]*\}`)

// stripTags removes WebVTT and SRT markup: <i>, <c.x>, <v Name>, {\an8}.
func stripTags(s string) string {
	s = tagRE.ReplaceAllString(s, "")
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&nbsp;", "\u00a0", "&lrm;", "\u200e", "&rlm;", "\u200f")
	return r.Replace(s)
}

// AddCaptions adds a caption track and returns its index. The first
// one added is shown.
func (p *Player) AddCaptions(t Track) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tracks = append(p.tracks, t)
	if p.track < 0 && len(p.tracks) == 1 {
		p.track = 0
	}
	return len(p.tracks) - 1
}

// Captions are the tracks added.
func (p *Player) Captions() []Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Track(nil), p.tracks...)
}

// SetCaptionTrack shows track i; -1 hides captions.
func (p *Player) SetCaptionTrack(i int) {
	p.mu.Lock()
	if i < -1 || i >= len(p.tracks) {
		i = -1
	}
	p.track = i
	p.mu.Unlock()
	p.invalidate()
}

// CaptionTrack is the track shown, or -1.
func (p *Player) CaptionTrack() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.track
}

// Caption is the text to show now; empty between cues.
func (p *Player) Caption() string {
	pos := p.Position()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.track < 0 {
		return ""
	}
	return cueAt(p.tracks[p.track].Cues, pos)
}

// cueAt joins the cues showing at pos.
func cueAt(cues []Cue, pos time.Duration) string {
	i := sort.Search(len(cues), func(i int) bool { return cues[i].Start > pos })
	var out []string
	// Cues may overlap: look back over those that started before pos.
	for j := i - 1; j >= 0 && j >= i-8; j-- {
		if pos < cues[j].End {
			out = append([]string{cues[j].Text}, out...)
		}
	}
	return strings.Join(out, "\n")
}
