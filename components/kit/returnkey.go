package kit

// SetReturnKeyDone makes the software keyboard's return key read Done
// (on false, its default), where the platform and Gio allow it: giode
// sets it with the giode CLI's Gio patches. Text fields call it while
// focused.
var SetReturnKeyDone func(done bool)

// SetAutofill tells the platform what the focused text field holds,
// for password managers and AutoFill (see input.Autofill's kinds):
// giode sets it with the giode CLI's Gio patches. Text fields call it
// while focused.
var SetAutofill func(kind int)
