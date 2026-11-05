package base

import (
	"runtime"
	"testing"
)

func TestKeyed_PersistsAndExpires(t *testing.T) {
	SetKeyedRetention(3)
	defer SetKeyedRetention(DefaultRetainFrames)
	made := 0
	get := func(k string) *int {
		return Keyed("test-scope", k, func() *int { made++; v := 0; return &v })
	}
	BeginFrame()
	a := get("a")
	*a = 7
	get("b")
	EndFrame()
	for i := 0; i < 5; i++ {
		BeginFrame()
		if got := get("a"); got != a || *got != 7 {
			t.Fatal("state for a not persisted")
		}
		EndFrame()
	}
	if made != 2 {
		t.Errorf("init calls = %d, want 2", made)
	}
	BeginFrame()
	get("b") // b expired after 3 unrequested frames: recreated
	EndFrame()
	if made != 3 {
		t.Errorf("init calls after b expired = %d, want 3", made)
	}
}

func TestKeyed_ScopesAndTypes(t *testing.T) {
	x := Keyed("s1", 1, func() string { return "one" })
	y := Keyed("s2", 1, func() string { return "two" })
	z := Keyed("s1", 1, func() int { return 3 }) // same key, other type: replaced
	if x != "one" || y != "two" || z != 3 {
		t.Errorf("got %v %v %v", x, y, z)
	}
	ForgetKeyed("s1", 1)
	if Keyed("s1", 1, func() int { return 4 }) != 4 {
		t.Error("ForgetKeyed didn't drop the entry")
	}
}

func TestKeyedCache(t *testing.T) {
	var evicted []string
	c := NewKeyedCache[string, int]().Retain(2).OnEvict(func(k string, _ int) { evicted = append(evicted, k) })
	BeginFrame()
	c.Get("keep", func() int { return 1 })
	c.Get("drop", func() int { return 2 })
	EndFrame()
	for i := 0; i < 3; i++ {
		BeginFrame()
		c.Get("keep", func() int { return 99 })
		EndFrame()
	}
	if v, ok := c.Peek("keep"); !ok || v != 1 {
		t.Errorf("keep = %v, %v", v, ok)
	}
	if _, ok := c.Peek("drop"); ok || len(evicted) != 1 || evicted[0] != "drop" {
		t.Errorf("drop not evicted: %v", evicted)
	}
	c.Delete("keep")
	if c.Len() != 0 || len(evicted) != 2 {
		t.Errorf("Delete: len %d, evicted %v", c.Len(), evicted)
	}
}

func TestKeyedCache_CollectedCacheLeavesSweepList(t *testing.T) {
	keyedCacheMu.Lock()
	before := len(keyedCaches)
	keyedCacheMu.Unlock()
	func() { _ = NewKeyedCache[int, int]() }()
	runtime.GC()
	BeginFrame()
	EndFrame()
	keyedCacheMu.Lock()
	after := len(keyedCaches)
	keyedCacheMu.Unlock()
	if after > before {
		t.Errorf("sweep list grew from %d to %d: collected cache kept", before, after)
	}
}
