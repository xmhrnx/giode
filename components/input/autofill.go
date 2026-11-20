package input

// AutofillKind is what a field holds, for password managers and the
// keyboard's AutoFill.
type AutofillKind int

const (
	AutofillNone AutofillKind = iota
	AutofillUsername
	AutofillPassword
	// AutofillNewPassword is a password being chosen: managers offer to
	// make a strong one and save it.
	AutofillNewPassword
	AutofillEmail
	// AutofillOneTimeCode is a code sent by text message or email: the
	// keyboard offers it as it arrives.
	AutofillOneTimeCode
	AutofillName
	AutofillPhone
	AutofillAddress
	AutofillPostalCode
)

// Autofill says what the field holds, so password managers (iCloud
// Keychain, Google Password Manager, 1Password, Bitwarden) and the
// keyboard can fill it. A Password field is AutofillPassword unless set
// otherwise. It works on iOS and Android in builds made with the
// Giode CLI. It returns i for chaining.
func (i *Input) Autofill(kind AutofillKind) *Input {
	i.autofill = kind
	return i
}

// autofillKind is what the platform is told while the field has focus.
func (i *Input) autofillKind() AutofillKind {
	if i.autofill == AutofillNone && i.editor.Mask != 0 {
		return AutofillPassword
	}
	return i.autofill
}
