package base

import "fmt"

// semanticID returns a stable ID for target, or a frame-order ID for
// nodes without one.
func semanticID(kind SemanticKind, target any) string {
	semanticMu.Lock()
	defer semanticMu.Unlock()
	if target != nil {
		if id, ok := semanticIDs[target]; ok {
			return id
		}
	}
	semanticSeq[kind]++
	id := fmt.Sprintf("%s-%d", kind, semanticSeq[kind])
	if target != nil {
		semanticIDs[target] = id
	} else {
		// Untargeted nodes are renumbered every frame.
		id = fmt.Sprintf("%s@%d", kind, len(*semantics))
		semanticSeq[kind]--
	}
	return id
}
