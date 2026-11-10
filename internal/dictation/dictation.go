// Package dictation connects giode.App.Dictate to package dictate,
// which holds the platform code, so that apps that don't dictate don't
// link the speech recognizer or record from the microphone: App Store
// review asks for purpose strings for what a binary links.
package dictation

// Linked is set when package dictate is imported.
var Linked bool

// Start, set by package dictate on iOS, starts dictating in lang (a BCP
// 47 tag, or "" for the device's language). emit receives each result;
// the final one comes last, once. stop ends listening early; the final
// result still follows.
var Start func(lang string, emit func(text string, final bool, err error)) (stop func(), err error)
