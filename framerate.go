package giode

import "time"

// SetMaxFPS caps the frame rate at fps frames a second; 0 draws at the
// display's refresh rate. See Config.MaxFPS. It may be called from any
// goroutine and applies from the next frame.
func (a *App) SetMaxFPS(fps int) {
	a.maxFPS.Store(int64(max(fps, 0)))
}

// MaxFPS returns the frame-rate cap, 0 for none.
func (a *App) MaxFPS() int {
	return int(a.maxFPS.Load())
}

// throttleFrame holds a frame back until 1/MaxFPS has passed since the
// last one started, and records when this one starts. With no cap it
// returns at once.
func (a *App) throttleFrame() {
	if wait := frameWait(a.lastFrame, time.Now(), int(a.maxFPS.Load())); wait > 0 {
		time.Sleep(wait)
	}
	a.lastFrame = time.Now()
}

// frameWait is how long a frame starting at now must wait for one that
// started at last to be 1/fps apart.
func frameWait(last, now time.Time, fps int) time.Duration {
	if fps <= 0 || last.IsZero() {
		return 0
	}
	interval := time.Second / time.Duration(fps)
	return interval - now.Sub(last)
}
