package box

import "time"

// hoverAnim is the transition state of a hoverable Box.
type hoverAnim struct {
	progress  float32
	animating bool
	lastFrame time.Time
}
