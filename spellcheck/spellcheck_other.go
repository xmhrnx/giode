//go:build !linux && !darwin && !windows

package spellcheck

func available(string) bool              { return false }
func check(string, string) []Misspelling { return nil }
func suggest(string, string) []string    { return nil }
func learn(string)                       {}
