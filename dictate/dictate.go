// Package dictate transcribes speech from the microphone, for a
// microphone button in a composer:
//
//	results, err := dictate.Start(ctx, app, "")
//	if err != nil {
//		return err
//	}
//	for t := range results {
//		field.SetText(t.Text)
//	}
//
// It is a package of its own so that only apps that dictate link the
// speech recognizer and record from the microphone: the App Store asks
// any app whose binary does for microphone and speech recognition
// purpose strings. Importing it also makes giode.App.Dictate work.
//
// Android (SpeechRecognizer; the package adds the microphone permission
// to the manifest) and iOS (SFSpeechRecognizer; set
// ios.usage.microphone and ios.usage.speech in giode.toml, the
// Info.plist's NSMicrophoneUsageDescription and
// NSSpeechRecognitionUsageDescription) are supported; elsewhere Start
// returns giode.ErrNotSupported.
package dictate

import (
	"context"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/internal/dictation"
)

// Transcript is one result: the text recognized so far, replaced by
// each later result, until the Final one.
type Transcript = giode.Transcript

func init() {
	dictation.Linked = true
}

// Start listens through the microphone and transcribes speech, asking
// for permission the first time. Results arrive on the channel as the
// user speaks - each holds the whole transcript so far - and the
// channel closes after the Final one, which comes when the user stops
// speaking. Cancelling ctx stops listening early; a final result still
// follows. lang is a BCP 47 tag ("en-US"), or "" for the device's
// language. It is a's Dictate.
func Start(ctx context.Context, a *giode.App, lang string) (<-chan Transcript, error) {
	return a.Dictate(ctx, lang)
}
