package giode

import (
	"context"
	"fmt"

	"github.com/xmhrnx/giode/internal/dictation"
)

// Transcript is one result of Dictate: the text recognized so far,
// replaced by each later result, until the Final one.
type Transcript struct {
	Text  string
	Final bool
	Err   error
}

// Dictate listens through the microphone and transcribes speech, for a
// microphone button in a composer. Results arrive on the channel as the
// user speaks - each holds the whole transcript so far - and the
// channel closes after the Final one, which comes when the user stops
// speaking. Cancelling ctx stops listening early; a final result still
// follows.
//
// It needs the github.com/xmhrnx/giode/dictate package imported (see
// it): the speech recognizer and microphone code live there, so apps
// that don't dictate don't carry them, nor face App Store questions
// about them. Without it Dictate returns an error wrapping
// ErrNotSupported.
//
// It uses the platform's speech recognizer, asking for permission the
// first time: Android's SpeechRecognizer, and SFSpeechRecognizer on iOS
// (the app's Info.plist needs NSMicrophoneUsageDescription and
// NSSpeechRecognitionUsageDescription, ios.usage.microphone and
// ios.usage.speech in giode.toml, or it returns an error wrapping
// ErrNotSupported; see docs/platform.md). Elsewhere it returns
// ErrNotSupported. lang is a BCP 47 tag ("en-US"), or "" for the
// device's language.
func (a *App) Dictate(ctx context.Context, lang string) (<-chan Transcript, error) {
	if !dictation.Linked {
		return nil, errDictateNotLinked
	}
	return a.dictate(ctx, lang)
}

var errDictateNotLinked = fmt.Errorf("%w: dictation needs the github.com/xmhrnx/giode/dictate package imported", ErrNotSupported)
