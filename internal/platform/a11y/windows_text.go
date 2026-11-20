//go:build windows

package a11y

import (
	"image"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/xmhrnx/giode/elements"
)

// UI Automation's Text pattern, for text fields and text: an
// ITextProvider on the node, and text ranges over its text that a screen
// reader moves by character, word or line to read what the caret passes
// and where it is. Ranges are COM objects of their own, kept from the
// collector while clients hold them.

var (
	iidTextProvider = windows.GUID{Data1: 0x3589c92c, Data2: 0x63f3, Data3: 0x4367, Data4: [8]byte{0x99, 0xbb, 0xad, 0xa6, 0x53, 0xb7, 0x7c, 0xf2}}
	iidTextRange    = windows.GUID{Data1: 0x5347ad7b, Data2: 0xc355, Data3: 0x46f8, Data4: [8]byte{0xaf, 0xf5, 0x90, 0x90, 0x33, 0x58, 0x2f, 0x63}}

	procUiaGetReservedNotSupportedValue = uiaCore.NewProc("UiaGetReservedNotSupportedValue")
)

const vtUnknown = 13

// uiaRange is a range of a node's text, [start, end) in runes.
type uiaRange struct {
	vtbl *[21]uintptr
	refs atomic.Int32
	b    *uiaBridge
	k    int32

	mu         sync.Mutex
	start, end int
}

func (b *uiaBridge) newRange(k int32, start, end int) unsafe.Pointer {
	r := &uiaRange{vtbl: vtTextRng, b: b, k: k, start: start, end: end}
	r.refs.Store(1)
	b.mu.Lock()
	if b.ranges == nil {
		b.ranges = map[*uiaRange]struct{}{}
	}
	b.ranges[r] = struct{}{}
	b.mu.Unlock()
	return unsafe.Pointer(r)
}

// rangeOf returns the range p points to, nil when p isn't one of ours.
func rangeOf(p unsafe.Pointer) *uiaRange {
	if p == nil || *(**[21]uintptr)(p) != vtTextRng {
		return nil
	}
	return (*uiaRange)(p)
}

func (r *uiaRange) release() uintptr {
	n := r.refs.Add(-1)
	if n <= 0 {
		r.b.mu.Lock()
		delete(r.b.ranges, r)
		r.b.mu.Unlock()
		return 0
	}
	return uintptr(n)
}

// a11yTextContent is the text a node's text interfaces hold.
func a11yTextContent(n elements.SemanticNode) string {
	if n.Kind == elements.SemanticInput {
		return n.Value
	}
	return a11yText(n)
}

// text returns the range's node and its text.
func (r *uiaRange) text() (elements.SemanticNode, []rune, bool) {
	n, _, _, ok := r.b.t.node(r.k)
	if !ok {
		return n, nil, false
	}
	return n, []rune(a11yTextContent(n)), true
}

// span returns the range clamped to the text.
func (r *uiaRange) span(t []rune) (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, e := max(0, min(r.start, len(t))), max(0, min(r.end, len(t)))
	return min(s, e), max(s, e)
}

func (r *uiaRange) set(s, e int) {
	r.mu.Lock()
	r.start, r.end = s, e
	r.mu.Unlock()
}

// safeArrayOfRanges returns a SAFEARRAY of IUnknown holding ranges, which
// it takes the caller's references to.
func safeArrayOfRanges(ranges ...unsafe.Pointer) uintptr {
	sa, _, _ := procSafeArrayCreateVector.Call(vtUnknown, 0, uintptr(len(ranges)))
	for i, p := range ranges {
		idx := int32(i)
		// SafeArrayPutElement takes its own reference.
		procSafeArrayPutElement.Call(sa, uintptr(unsafe.Pointer(&idx)), uintptr(p))
		rangeOf(p).release()
	}
	return sa
}

func initUIATextVtables() {
	u := unknown(offText)
	textProvider := &[9]uintptr{u[0], u[1], u[2],
		// GetSelection
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			o := nodeOf(this, offText)
			n, ok := o.node()
			if !ok {
				return uiaENotAvailable
			}
			if start, end, ok := a11ySelection(n); ok {
				*out = safeArrayOfRanges(o.b.newRange(o.k, start, end))
			} else {
				*out = safeArrayOfRanges()
			}
			return sOK
		}),
		// GetVisibleRanges
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			o := nodeOf(this, offText)
			n, ok := o.node()
			if !ok {
				return uiaENotAvailable
			}
			*out = safeArrayOfRanges(o.b.newRange(o.k, 0, len([]rune(a11yTextContent(n)))))
			return sOK
		}),
		// RangeFromChild: the text has no children; the whole of it.
		windows.NewCallback(func(this, child unsafe.Pointer, out *unsafe.Pointer) uintptr {
			return nodeOf(this, offText).documentRange(out)
		}),
		// RangeFromPoint: its point can't be read (see the callback); the
		// start.
		uiaRangeFromPointCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			o := nodeOf(this, offText)
			*out = o.b.newRange(o.k, 0, 0)
			return sOK
		}),
		// get_DocumentRange
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			return nodeOf(this, offText).documentRange(out)
		}),
		// get_SupportedTextSelection: one selection.
		windows.NewCallback(func(this unsafe.Pointer, out *int32) uintptr {
			*out = 1
			return sOK
		}),
	}

	rangeUnknown := [3]uintptr{
		windows.NewCallback(func(this unsafe.Pointer, riid *windows.GUID, out *unsafe.Pointer) uintptr {
			*out = nil
			if *riid != iidIUnknown && *riid != iidTextRange {
				return eNoInterface
			}
			rangeOf(this).refs.Add(1)
			*out = this
			return sOK
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return uintptr(rangeOf(this).refs.Add(1))
		}),
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			return rangeOf(this).release()
		}),
	}
	var findAttribute uintptr
	if unsafe.Sizeof(uintptr(0)) == 8 {
		// The VARIANT it takes by value goes by reference on 64-bit
		// Windows.
		findAttribute = windows.NewCallback(func(this unsafe.Pointer, attr, val, backward uintptr, out *unsafe.Pointer) uintptr {
			*out = nil
			return sOK
		})
	} else {
		// On 32-bit Windows, on the stack: four words.
		findAttribute = windows.NewCallback(func(this unsafe.Pointer, attr, v0, v1, v2, v3, backward uintptr, out *unsafe.Pointer) uintptr {
			*out = nil
			return sOK
		})
	}
	textRange := &[21]uintptr{rangeUnknown[0], rangeUnknown[1], rangeUnknown[2],
		// Clone
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			r := rangeOf(this)
			r.mu.Lock()
			s, e := r.start, r.end
			r.mu.Unlock()
			*out = r.b.newRange(r.k, s, e)
			return sOK
		}),
		// Compare
		windows.NewCallback(func(this, other unsafe.Pointer, out *int32) uintptr {
			r, o := rangeOf(this), rangeOf(other)
			*out = 0
			if o == nil {
				return eInvalidArg
			}
			_, t, _ := r.text()
			s1, e1 := r.span(t)
			s2, e2 := o.span(t)
			*out = boolInt(r.k == o.k && s1 == s2 && e1 == e2)
			return sOK
		}),
		// CompareEndpoints
		windows.NewCallback(func(this unsafe.Pointer, ep uintptr, other unsafe.Pointer, otherEp uintptr, out *int32) uintptr {
			r, o := rangeOf(this), rangeOf(other)
			*out = 0
			if o == nil {
				return eInvalidArg
			}
			_, t, _ := r.text()
			a := endpoint(r, t, int32(ep))
			b := endpoint(o, t, int32(otherEp))
			*out = int32(a - b)
			return sOK
		}),
		// ExpandToEnclosingUnit
		windows.NewCallback(func(this unsafe.Pointer, unit uintptr) uintptr {
			r := rangeOf(this)
			_, t, ok := r.text()
			if !ok {
				return uiaENotAvailable
			}
			s, _ := r.span(t)
			r.set(enclosingUnit(t, s, textUnit(int32(unit))))
			return sOK
		}),
		findAttribute,
		// FindText
		windows.NewCallback(func(this unsafe.Pointer, text *uint16, backward, ignoreCase uintptr, out *unsafe.Pointer) uintptr {
			r := rangeOf(this)
			*out = nil
			_, t, ok := r.text()
			if !ok {
				return uiaENotAvailable
			}
			s, e := r.span(t)
			hay, needle := string(t[s:e]), windows.UTF16PtrToString(text)
			if int32(ignoreCase) != 0 {
				hay, needle = strings.ToLower(hay), strings.ToLower(needle)
			}
			var i int
			if int32(backward) != 0 {
				i = strings.LastIndex(hay, needle)
			} else {
				i = strings.Index(hay, needle)
			}
			if i >= 0 && needle != "" {
				at := s + len([]rune(hay[:i]))
				*out = r.b.newRange(r.k, at, at+len([]rune(needle)))
			}
			return sOK
		}),
		// GetAttributeValue: none are reported.
		windows.NewCallback(func(this unsafe.Pointer, attr uintptr, out *variant) uintptr {
			var unk uintptr
			procUiaGetReservedNotSupportedValue.Call(uintptr(unsafe.Pointer(&unk)))
			out.vt, out.val = vtUnknown, uint64(unk)
			return sOK
		}),
		// GetBoundingRectangles
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			r := rangeOf(this)
			*out = 0
			n, t, ok := r.text()
			if !ok {
				return uiaENotAvailable
			}
			s, e := r.span(t)
			box := r.b.textBox(n, len(t), s, e)
			sa, _, _ := procSafeArrayCreateVector.Call(vtR8, 0, 4)
			for i, v := range [4]float64{float64(box.Min.X), float64(box.Min.Y), float64(box.Dx()), float64(box.Dy())} {
				idx := int32(i)
				procSafeArrayPutElement.Call(sa, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&v)))
			}
			*out = sa
			return sOK
		}),
		// GetEnclosingElement
		windows.NewCallback(func(this unsafe.Pointer, out *unsafe.Pointer) uintptr {
			r := rangeOf(this)
			*out = r.b.provider(r.k, offSimple)
			return sOK
		}),
		// GetText
		windows.NewCallback(func(this unsafe.Pointer, maxLen uintptr, out *uintptr) uintptr {
			r := rangeOf(this)
			_, t, ok := r.text()
			if !ok {
				*out = bstr("")
				return uiaENotAvailable
			}
			s, e := r.span(t)
			if m := int(int32(maxLen)); m >= 0 && e-s > m {
				e = s + m
			}
			*out = bstr(string(t[s:e]))
			return sOK
		}),
		// Move
		windows.NewCallback(func(this unsafe.Pointer, unit, count uintptr, out *int32) uintptr {
			r := rangeOf(this)
			*out = 0
			_, t, ok := r.text()
			if !ok {
				return uiaENotAvailable
			}
			u, c := textUnit(int32(unit)), int(int32(count))
			s, e := r.span(t)
			if c == 0 {
				return sOK
			}
			pos, moved := moveBy(t, s, u, c)
			if s == e {
				r.set(pos, pos)
			} else {
				r.set(enclosingUnit(t, pos, u))
			}
			*out = int32(moved)
			return sOK
		}),
		// MoveEndpointByUnit
		windows.NewCallback(func(this unsafe.Pointer, ep, unit, count uintptr, out *int32) uintptr {
			r := rangeOf(this)
			*out = 0
			_, t, ok := r.text()
			if !ok {
				return uiaENotAvailable
			}
			s, e := r.span(t)
			if int32(ep) == 0 {
				pos, moved := moveBy(t, s, textUnit(int32(unit)), int(int32(count)))
				r.set(pos, max(pos, e))
				*out = int32(moved)
			} else {
				pos, moved := moveBy(t, e, textUnit(int32(unit)), int(int32(count)))
				r.set(min(s, pos), pos)
				*out = int32(moved)
			}
			return sOK
		}),
		// MoveEndpointByRange
		windows.NewCallback(func(this unsafe.Pointer, ep uintptr, other unsafe.Pointer, otherEp uintptr) uintptr {
			r, o := rangeOf(this), rangeOf(other)
			if o == nil {
				return eInvalidArg
			}
			_, t, _ := r.text()
			s, e := r.span(t)
			pos := endpoint(o, t, int32(otherEp))
			if int32(ep) == 0 {
				r.set(pos, max(pos, e))
			} else {
				r.set(min(s, pos), pos)
			}
			return sOK
		}),
		// Select
		windows.NewCallback(func(this unsafe.Pointer) uintptr {
			r := rangeOf(this)
			_, t, ok := r.text()
			if !ok || r.b.a == nil {
				return uiaENotAvailable
			}
			s, e := r.span(t)
			if !r.b.t.perform(r.b.a, r.k, a11ySetSelection, itoa(s)+","+itoa(e)) {
				return uiaEInvalidOp
			}
			return sOK
		}),
		// AddToSelection, RemoveFromSelection: one selection only.
		windows.NewCallback(func(this unsafe.Pointer) uintptr { return uiaEInvalidOp }),
		windows.NewCallback(func(this unsafe.Pointer) uintptr { return uiaEInvalidOp }),
		// ScrollIntoView
		windows.NewCallback(func(this unsafe.Pointer, alignTop uintptr) uintptr { return sOK }),
		// GetChildren
		windows.NewCallback(func(this unsafe.Pointer, out *uintptr) uintptr {
			*out = safeArrayOfRanges()
			return sOK
		}),
	}
	vtText, vtTextRng = textProvider, textRange
}

const eInvalidArg = 0x80070057

// endpoint returns r's start (ep 0) or end, clamped to t.
func endpoint(r *uiaRange, t []rune, ep int32) int {
	s, e := r.span(t)
	if ep == 0 {
		return s
	}
	return e
}

func (o *uiaNode) documentRange(out *unsafe.Pointer) uintptr {
	n, ok := o.node()
	if !ok {
		*out = nil
		return uiaENotAvailable
	}
	*out = o.b.newRange(o.k, 0, len([]rune(a11yTextContent(n))))
	return sOK
}

// textBox estimates where [s, e) of a node's count runes is on the
// screen: glyph positions aren't in the tree, so a share of the node's
// width; an empty range is a caret-wide line.
func (b *uiaBridge) textBox(n elements.SemanticNode, count, s, e int) image.Rectangle {
	r := n.Bounds
	pt := [2]int32{int32(r.Min.X), int32(r.Min.Y)}
	procClientToScreen.Call(b.hwnd, uintptr(unsafe.Pointer(&pt)))
	r = r.Add(image.Pt(int(pt[0])-r.Min.X, int(pt[1])-r.Min.Y))
	if count == 0 {
		return image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y)
	}
	x0 := r.Min.X + r.Dx()*s/count
	x1 := r.Min.X + r.Dx()*e/count
	if x1 <= x0 {
		x1 = x0 + 1
	}
	return image.Rect(x0, r.Min.Y, x1, r.Max.Y)
}

// raiseText reports focused node k's caret moving and text changing.
func (b *uiaBridge) raiseText(k int32, n elements.SemanticNode) {
	if !n.HasCaret {
		return
	}
	b.mu.Lock()
	caret := [3]int{int(k), n.Caret, n.Anchor}
	moved := caret != b.caret
	typed := int(k) == b.caret[0] && n.Value != b.text
	b.caret, b.text = caret, n.Value
	b.mu.Unlock()
	if typed {
		b.raise(k, evTextChanged)
	}
	if moved {
		b.raise(k, evTextSelection)
	}
}
