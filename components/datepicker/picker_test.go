package datepicker

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"

	"github.com/xmhrnx/giode/datetime"
)

func TestDatePickerTyping(t *testing.T) {
	var got time.Time
	p := New().Locale(datetime.NewLocale("en-US")).Zone(time.UTC).OnChange(func(d time.Time) { got = d })
	h := newHarness(p.Layout)
	h.frame()
	p.Field().Focus()
	h.frame()
	h.frame()
	h.r.Queue(key.EditEvent{Text: "11/20/2026"})
	h.frame()
	h.frame()
	if !got.Equal(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("typed date = %s (field %q)", got, p.Field().Text())
	}
	// Leaving it formatted; an invalid date marks the field.
	p.Field().SetText("13/45/2026")
	p.commit()
	if !p.Value().Equal(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("an invalid date changed the value: %s", p.Value())
	}
	p.Field().SetText("")
	p.commit()
	if !p.Value().IsZero() || !got.IsZero() {
		t.Errorf("emptied field kept %s", p.Value())
	}
}

func TestDatePickerCalendar(t *testing.T) {
	var got time.Time
	p := New().Locale(datetime.NewLocale("en-US")).Zone(time.UTC).OnChange(func(d time.Time) { got = d })
	p.SetValue(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC))
	h := newHarness(p.Layout)
	h.frame()
	p.button.Click()
	h.frame()
	h.frame()
	if !p.Open() {
		t.Fatal("the button didn't open the calendar")
	}
	// The calendar's grid, where the portal placed it: find the 20th's
	// cell from its tracked position.
	c := p.cal
	if len(c.grids) == 0 {
		t.Fatal("calendar not laid out")
	}
	at := c.origin
	pt := cellCenter(c, time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC))
	h.click(f32.Pt(pt.X+float32(at.X), pt.Y+float32(at.Y)))
	if !got.Equal(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("picked %s", got)
	}
	if p.Open() {
		t.Error("picking didn't close the calendar")
	}
	if p.Field().Text() != "11/20/2026" {
		t.Errorf("field shows %q", p.Field().Text())
	}
	_ = image.Point{}
}

func TestTimePicker(t *testing.T) {
	var got TimeOfDay
	var ok bool
	p := NewTime().Locale(datetime.NewLocale("en-US")).Step(15 * time.Minute).OnChange(func(t TimeOfDay, k bool) { got, ok = t, k })
	h := newHarness(p.Layout)
	h.frame()
	p.Field().Focus()
	h.frame()
	h.frame()
	h.r.Queue(key.EditEvent{Text: "3:45pm"})
	h.frame()
	h.frame()
	if !ok || got != (TimeOfDay{15, 45}) {
		t.Fatalf("typed = %v %v", got, ok)
	}
	p.commit()
	if p.Field().Text() != "3:45 PM" {
		t.Errorf("formatted as %q", p.Field().Text())
	}
	// The list: 96 slots at 15 minutes; clicking one picks it.
	p.openList()
	h.frame()
	h.frame()
	if len(p.slots) != 96 {
		t.Fatalf("%d slots", len(p.slots))
	}
	p.slots[40].Click() // 10:00
	h.frame()
	if got != (TimeOfDay{10, 0}) || p.Open() {
		t.Errorf("picked %v, open %v", got, p.Open())
	}
}

func TestDateTimeField(t *testing.T) {
	ny, _ := datetime.LoadZone("America/New_York")
	f := NewDateTime().Locale(datetime.NewLocale("en-US")).Zone(ny)
	want := time.Date(2026, 3, 8, 9, 30, 0, 0, ny) // the day clocks go forward
	f.SetValue(want)
	got, ok := f.Value()
	if !ok || !got.Equal(want) {
		t.Errorf("value = %s, %v", got, ok)
	}
	if f.Date().Field().Text() != "3/8/2026" || f.Time().Field().Text() != "9:30 AM" {
		t.Errorf("fields %q %q", f.Date().Field().Text(), f.Time().Field().Text())
	}
}
