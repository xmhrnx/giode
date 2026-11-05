package base

// OutOfFlow is implemented by elements that paint over the window and
// take no space where they're placed: Dialog, Sheet and Drawer. A box
// lays them out apart from its other children, so its Gap and Justify
// leave them out.
type OutOfFlow interface {
	Element
	OutOfFlow()
}
