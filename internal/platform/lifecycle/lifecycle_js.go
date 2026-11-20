package lifecycle

// The web reports nothing.
const FromWindowState = false

func Watch(background, foreground func()) error { return nil }
