package tabs

import (
	"testing"

	"github.com/xmhrnx/giode/elements"
	"github.com/xmhrnx/giode/styles"
)

func view() elements.Element { return elements.Box(styles.Styles{}) }

// Tabs change after creation; the selection stays on its tab.
func TestTabsAtRuntime(t *testing.T) {
	tb := New(styles.Styles{}, Tab{Label: "A", View: view}, Tab{Label: "B", View: view})
	tb.Add(Tab{Label: "C", View: view})
	tb.SetSelected(2)
	tb.Remove(0)
	if tb.Len() != 2 || tb.Selected() != 1 || tb.tabs[1].Label != "C" {
		t.Fatalf("after removing A: %d tabs, selected %d", tb.Len(), tb.Selected())
	}
	tb.SetLabel(1, "C2")
	tb.Remove(1)
	if tb.Selected() != 0 || tb.tabs[0].Label != "B" || len(tb.clicks) != 1 {
		t.Fatalf("after removing the selected last tab: selected %d", tb.Selected())
	}
	tb.SetTabs(Tab{Label: "X", View: view}, Tab{Label: "Y", View: view}, Tab{Label: "Z", View: view})
	if tb.Len() != 3 || len(tb.clicks) != 3 {
		t.Errorf("SetTabs: %d tabs, %d clickables", tb.Len(), len(tb.clicks))
	}
}
