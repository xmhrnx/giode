package base

// retainStack collects, for caches recording their subtree, the keyed
// lookups to repeat on frames they replay instead.
var retainStack []*[]func()

func pushRetainScope(s *[]func()) (pop func()) {
	retainStack = append(retainStack, s)
	n := len(retainStack)
	return func() { retainStack = retainStack[:n-1] }
}

// retainKeyed registers touch with the innermost recording cache.
func retainKeyed(touch func()) {
	if n := len(retainStack); n > 0 {
		s := retainStack[n-1]
		*s = append(*s, touch)
	}
}

// RetainKeyed registers touch with the innermost recording cache. It
// is retainKeyed for element packages replayed by a MemoCache.
func RetainKeyed(touch func()) {
	retainKeyed(touch)
}
