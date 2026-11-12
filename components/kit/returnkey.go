package kit

// SetReturnKeyDone makes the software keyboard's return key read Done
// (on false, its default), where the platform and Gio allow it: giode
// sets it with the giode CLI's Gio patches. Text fields call it while
// focused.
var SetReturnKeyDone func(done bool)
