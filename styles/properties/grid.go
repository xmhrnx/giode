package properties

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Grid layout (Display: DisplayGrid), after CSS grid: the container's
// GridColumns and GridRows size its tracks, and each child's
// GridColumn and GridRow place it, or it's placed in the next free
// cells in reading order.

// DisplayGrid lays a Box's children out on a grid.
const DisplayGrid Display = "grid"

// GridTracks sizes a grid's columns or rows, as CSS's
// grid-template-columns does, sizes in dp:
//
//	"200 1fr 2fr"                          200 dp, then a third and two thirds of the rest
//	"auto 1fr"                             a column as wide as its content, then the rest
//	"repeat(3, 1fr)"                       three equal columns
//	"minmax(160, 1fr) 2fr"                 a share of at least 160 dp
//	"repeat(auto-fill, minmax(220, 1fr))"  as many columns of at least 220 dp as fit
//
// Rows past those given are auto.
type GridTracks string

// TrackKind is a grid track's sizing.
type TrackKind uint8

const (
	// TrackAuto is as wide (or tall) as the track's content.
	TrackAuto TrackKind = iota
	// TrackFixed is Size dp.
	TrackFixed
	// TrackFr takes Size shares of the space left by the others.
	TrackFr
)

// Track is one parsed grid track. Min is a minmax minimum in dp.
type Track struct {
	Kind TrackKind
	Size float32
	Min  float32
}

// ParsedTracks is a GridTracks spec read: the tracks, or with AutoFill
// set, the one track repeated as many times as fit.
type ParsedTracks struct {
	Tracks   []Track
	AutoFill bool
}

var trackCache sync.Map // GridTracks -> ParsedTracks

// Parse reads the spec, once per spec; a malformed one reads as no
// tracks, a single auto column.
func (g GridTracks) Parse() ParsedTracks {
	if g == "" {
		return ParsedTracks{}
	}
	if p, ok := trackCache.Load(g); ok {
		return p.(ParsedTracks)
	}
	p, err := parseTracks(string(g))
	if err != nil {
		p = ParsedTracks{}
	}
	trackCache.Store(g, p)
	return p
}

// Check reports a malformed spec.
func (g GridTracks) Check() error {
	_, err := parseTracks(string(g))
	return err
}

func parseTracks(s string) (ParsedTracks, error) {
	var out ParsedTracks
	toks, err := splitTracks(s)
	if err != nil {
		return out, err
	}
	for _, tok := range toks {
		if inner, ok := call(tok, "repeat"); ok {
			args := splitArgs(inner)
			if len(args) != 2 {
				return out, fmt.Errorf("grid: %q: repeat takes a count and tracks", tok)
			}
			sub, err := parseTracks(args[1])
			if err != nil {
				return out, err
			}
			if a := strings.TrimSpace(args[0]); a == "auto-fill" || a == "auto-fit" {
				if len(sub.Tracks) != 1 || len(toks) != 1 {
					return out, fmt.Errorf("grid: %q: auto-fill repeats one track, alone", tok)
				}
				out.Tracks, out.AutoFill = sub.Tracks, true
				continue
			}
			n, err := strconv.Atoi(strings.TrimSpace(args[0]))
			if err != nil || n < 1 || n > 1000 {
				return out, fmt.Errorf("grid: %q: bad count", tok)
			}
			for range n {
				out.Tracks = append(out.Tracks, sub.Tracks...)
			}
			continue
		}
		t, err := parseTrack(tok)
		if err != nil {
			return out, err
		}
		out.Tracks = append(out.Tracks, t)
	}
	return out, nil
}

func parseTrack(tok string) (Track, error) {
	if inner, ok := call(tok, "minmax"); ok {
		args := splitArgs(inner)
		if len(args) != 2 {
			return Track{}, fmt.Errorf("grid: %q: minmax takes two sizes", tok)
		}
		lo, err := parseTrack(strings.TrimSpace(args[0]))
		if err != nil || lo.Kind != TrackFixed {
			return Track{}, fmt.Errorf("grid: %q: minmax's minimum is a size in dp", tok)
		}
		hi, err := parseTrack(strings.TrimSpace(args[1]))
		if err != nil {
			return Track{}, err
		}
		if hi.Kind == TrackFixed {
			hi.Size = max(hi.Size, lo.Size)
		}
		hi.Min = lo.Size
		return hi, nil
	}
	switch {
	case tok == "auto":
		return Track{Kind: TrackAuto}, nil
	case strings.HasSuffix(tok, "fr"):
		v, err := strconv.ParseFloat(strings.TrimSuffix(tok, "fr"), 32)
		if err != nil || v <= 0 {
			return Track{}, fmt.Errorf("grid: bad track %q", tok)
		}
		return Track{Kind: TrackFr, Size: float32(v)}, nil
	}
	num := strings.TrimSuffix(strings.TrimSuffix(tok, "px"), "dp")
	v, err := strconv.ParseFloat(num, 32)
	if err != nil || v < 0 {
		return Track{}, fmt.Errorf("grid: bad track %q", tok)
	}
	return Track{Kind: TrackFixed, Size: float32(v)}, nil
}

// splitTracks splits s at spaces outside parentheses.
func splitTracks(s string) ([]string, error) {
	var out []string
	depth, start := 0, -1
	for i, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("grid: %q: unbalanced )", s)
			}
		case (r == ' ' || r == '\t') && depth == 0:
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("grid: %q: unbalanced (", s)
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out, nil
}

// call returns the arguments of name(...) when tok is that call.
func call(tok, name string) (string, bool) {
	if strings.HasPrefix(tok, name+"(") && strings.HasSuffix(tok, ")") {
		return tok[len(name)+1 : len(tok)-1], true
	}
	return "", false
}

// splitArgs splits at commas outside parentheses.
func splitArgs(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

// GridSpan places a child on a grid's columns or rows: from Start (the
// first is 1; 0 places it in the next free cells) across Span tracks
// (0 is 1).
type GridSpan struct {
	Start, Span int
}

// Span places a child across n tracks, wherever there's room.
func Span(n int) GridSpan { return GridSpan{Span: n} }

// Cell places a child from track start (1-based) across span tracks.
func Cell(start, span int) GridSpan { return GridSpan{Start: start, Span: span} }
