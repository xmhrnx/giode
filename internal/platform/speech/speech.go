// Package speech transcribes the microphone with the platform's speech
// recognizer.
package speech

// Result is the text recognized so far, replaced by each later result,
// until the Final one.
type Result struct {
	Text  string
	Final bool
	Err   error
}
