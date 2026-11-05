package base

import (
	"github.com/xmhrnx/giode/components/kit"
	"github.com/xmhrnx/giode/internal/textrender"
)

// frameNumber counts frames; BeginFrame advances it. Keyed memos not
// used during a frame are dropped at the next EndFrame.
var (
	frameNumber uint64
	memos       = map[any]*MemoCache{}
)

// FrameNumber returns the current frame number, which BeginFrame
// advances. Element packages use it for state that is reset every
// frame.
func FrameNumber() uint64 {
	return frameNumber
}

// BeginFrame starts a new frame. The app calls it before laying out
// the view; component authors don't need to.
func BeginFrame() {
	frameNumber++
	kit.NextFrame()
	textrender.Tick()
	clear(probes)
	probes = probes[:0]
}

// EndFrame finishes a frame, releasing keyed memos (see Memo) that
// weren't laid out during it. The app calls it after every frame.
func EndFrame() {
	for k, c := range memos {
		if c.alive != frameNumber {
			delete(memos, k)
		}
	}
	sweepKeyed()
}
