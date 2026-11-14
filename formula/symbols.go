package formula

// class is an atom's spacing class, as TeX has them.
type class uint8

const (
	clsOrd class = iota
	clsOp
	clsBin
	clsRel
	clsOpen
	clsClose
	clsPunct
)

// sym is a symbol a command or character stands for.
type sym struct {
	s   string
	cls class
}

// symbols are the commands that stand for one symbol.
var symbols = map[string]sym{
	// Greek, lowercase (set in italic) and uppercase (upright).
	"alpha": {"α", clsOrd}, "beta": {"β", clsOrd}, "gamma": {"γ", clsOrd}, "delta": {"δ", clsOrd},
	"epsilon": {"ϵ", clsOrd}, "varepsilon": {"ε", clsOrd}, "zeta": {"ζ", clsOrd}, "eta": {"η", clsOrd},
	"theta": {"θ", clsOrd}, "vartheta": {"ϑ", clsOrd}, "iota": {"ι", clsOrd}, "kappa": {"κ", clsOrd},
	"lambda": {"λ", clsOrd}, "mu": {"μ", clsOrd}, "nu": {"ν", clsOrd}, "xi": {"ξ", clsOrd},
	"omicron": {"ο", clsOrd}, "pi": {"π", clsOrd}, "varpi": {"ϖ", clsOrd}, "rho": {"ρ", clsOrd},
	"varrho": {"ϱ", clsOrd}, "sigma": {"σ", clsOrd}, "varsigma": {"ς", clsOrd}, "tau": {"τ", clsOrd},
	"upsilon": {"υ", clsOrd}, "phi": {"ϕ", clsOrd}, "varphi": {"φ", clsOrd}, "chi": {"χ", clsOrd},
	"psi": {"ψ", clsOrd}, "omega": {"ω", clsOrd},
	"Gamma": {"Γ", clsOrd}, "Delta": {"Δ", clsOrd}, "Theta": {"Θ", clsOrd}, "Lambda": {"Λ", clsOrd},
	"Xi": {"Ξ", clsOrd}, "Pi": {"Π", clsOrd}, "Sigma": {"Σ", clsOrd}, "Upsilon": {"Υ", clsOrd},
	"Phi": {"Φ", clsOrd}, "Psi": {"Ψ", clsOrd}, "Omega": {"Ω", clsOrd},

	// Letter-like and other ordinary symbols.
	"nabla": {"∇", clsOrd}, "partial": {"∂", clsOrd}, "hbar": {"ℏ", clsOrd}, "ell": {"ℓ", clsOrd},
	"infty": {"∞", clsOrd}, "emptyset": {"∅", clsOrd}, "varnothing": {"∅", clsOrd}, "angle": {"∠", clsOrd},
	"degree": {"°", clsOrd}, "prime": {"′", clsOrd}, "forall": {"∀", clsOrd}, "exists": {"∃", clsOrd},
	"neg": {"¬", clsOrd}, "Re": {"ℜ", clsOrd}, "Im": {"ℑ", clsOrd}, "aleph": {"ℵ", clsOrd},
	"ldots": {"…", clsOrd}, "dots": {"…", clsOrd}, "cdots": {"⋯", clsOrd}, "vdots": {"⋮", clsOrd},
	"ddots": {"⋱", clsOrd}, "triangle": {"△", clsOrd}, "square": {"□", clsOrd}, "star": {"⋆", clsOrd},
	"dagger": {"†", clsOrd}, "top": {"⊤", clsOrd}, "bot": {"⊥", clsOrd},
	"|": {"‖", clsOrd}, "backslash": {"∖", clsOrd},
	"{": {"{", clsOpen}, "}": {"}", clsClose}, "%": {"%", clsOrd}, "$": {"$", clsOrd},
	"#": {"#", clsOrd}, "&": {"&", clsOrd}, "_": {"_", clsOrd},
	"langle": {"⟨", clsOpen}, "rangle": {"⟩", clsClose}, "lfloor": {"⌊", clsOpen}, "rfloor": {"⌋", clsClose},
	"lceil": {"⌈", clsOpen}, "rceil": {"⌉", clsClose}, "lvert": {"|", clsOpen}, "rvert": {"|", clsClose},
	"lVert": {"‖", clsOpen}, "rVert": {"‖", clsClose},

	// Binary operations.
	"pm": {"±", clsBin}, "mp": {"∓", clsBin}, "times": {"×", clsBin}, "cdot": {"⋅", clsBin},
	"div": {"÷", clsBin}, "ast": {"∗", clsBin}, "circ": {"∘", clsBin}, "bullet": {"∙", clsBin},
	"cup": {"∪", clsBin}, "cap": {"∩", clsBin}, "wedge": {"∧", clsBin}, "land": {"∧", clsBin},
	"vee": {"∨", clsBin}, "lor": {"∨", clsBin}, "oplus": {"⊕", clsBin}, "otimes": {"⊗", clsBin},
	"setminus": {"∖", clsBin},

	// Relations and arrows.
	"leq": {"≤", clsRel}, "le": {"≤", clsRel}, "geq": {"≥", clsRel}, "ge": {"≥", clsRel},
	"neq": {"≠", clsRel}, "ne": {"≠", clsRel}, "approx": {"≈", clsRel}, "equiv": {"≡", clsRel},
	"propto": {"∝", clsRel}, "sim": {"∼", clsRel}, "simeq": {"≃", clsRel}, "cong": {"≅", clsRel},
	"ll": {"≪", clsRel}, "gg": {"≫", clsRel}, "lesssim": {"≲", clsRel}, "gtrsim": {"≳", clsRel},
	"in": {"∈", clsRel}, "notin": {"∉", clsRel}, "ni": {"∋", clsRel}, "subset": {"⊂", clsRel},
	"supset": {"⊃", clsRel}, "subseteq": {"⊆", clsRel}, "supseteq": {"⊇", clsRel},
	"perp": {"⊥", clsRel}, "parallel": {"∥", clsRel}, "mid": {"∣", clsRel}, "doteq": {"≐", clsRel},
	"to": {"→", clsRel}, "rightarrow": {"→", clsRel}, "leftarrow": {"←", clsRel}, "gets": {"←", clsRel},
	"leftrightarrow": {"↔", clsRel}, "Rightarrow": {"⇒", clsRel}, "Leftarrow": {"⇐", clsRel},
	"Leftrightarrow": {"⇔", clsRel}, "implies": {"⟹", clsRel}, "impliedby": {"⟸", clsRel},
	"iff": {"⟺", clsRel}, "mapsto": {"↦", clsRel}, "longrightarrow": {"⟶", clsRel},
	"longleftarrow": {"⟵", clsRel}, "uparrow": {"↑", clsRel}, "downarrow": {"↓", clsRel},
	"rightleftharpoons": {"⇌", clsRel}, "coloneqq": {"≔", clsRel}, "triangleq": {"≜", clsRel},

	// Punctuation.
	"colon": {":", clsPunct},
}

// bigOps are the large operators: whether they take their limits above
// and below in display style (sums do, integrals don't).
var bigOps = map[string]struct {
	s      string
	limits bool
}{
	"sum": {"∑", true}, "prod": {"∏", true}, "coprod": {"∐", true}, "bigcup": {"⋃", true},
	"bigcap": {"⋂", true}, "bigoplus": {"⨁", true}, "bigotimes": {"⨂", true},
	"int": {"∫", false}, "iint": {"∬", false}, "iiint": {"∭", false}, "oint": {"∮", false},
	"oiint": {"∯", false},
}

// functions are set upright; those with limits take them below in
// display style.
var functions = map[string]bool{
	"sin": false, "cos": false, "tan": false, "cot": false, "sec": false, "csc": false,
	"arcsin": false, "arccos": false, "arctan": false, "sinh": false, "cosh": false, "tanh": false,
	"coth": false, "exp": false, "ln": false, "log": false, "lg": false, "det": true, "dim": false,
	"ker": false, "deg": false, "arg": false, "gcd": true, "Pr": true, "hom": false,
	"lim": true, "limsup": true, "liminf": true, "max": true, "min": true, "sup": true, "inf": true,
}

// charClass is the class of a plain character.
func charClass(r rune) class {
	switch r {
	case '+', '*', '±', '∓', '×', '⋅', '÷', '∘', '∪', '∩':
		return clsBin
	case '-', '−':
		return clsBin
	case '=', '<', '>', ':', '≤', '≥', '≠', '≈', '≡', '∝', '∼', '→', '←', '⇒', '⇐', '⇔', '∈', '⊂', '⊃':
		return clsRel
	case '(', '[', '⟨':
		return clsOpen
	case ')', ']', '⟩', '!', '?':
		return clsClose
	case ',', ';':
		return clsPunct
	}
	return clsOrd
}

// spaces are named spaces, in em.
var spaces = map[string]float32{
	",": 3.0 / 18, "thinspace": 3.0 / 18, ":": 4.0 / 18, ">": 4.0 / 18, "medspace": 4.0 / 18,
	";": 5.0 / 18, "thickspace": 5.0 / 18, "!": -3.0 / 18, "negthinspace": -3.0 / 18,
	"quad": 1, "qquad": 2, " ": 4.0 / 18, "enspace": 0.5,
}

// italic maps a letter to its math italic form, as TeX sets variables.
func italic(r rune) rune {
	switch {
	case r == 'h':
		return 'ℎ'
	case r >= 'a' && r <= 'z':
		return 0x1D44E + (r - 'a')
	case r >= 'A' && r <= 'Z':
		return 0x1D434 + (r - 'A')
	case r >= 'α' && r <= 'ω':
		return 0x1D6FC + (r - 'α')
	}
	switch r {
	case 'ϵ':
		return 0x1D716
	case 'ϑ':
		return 0x1D717
	case 'ϕ':
		return 0x1D719
	case 'ϱ':
		return 0x1D71A
	case 'ϖ':
		return 0x1D71B
	case '∂':
		return 0x1D715
	}
	return r
}

// bold maps a letter or digit to its math bold form.
func bold(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return 0x1D400 + (r - 'A')
	case r >= 'a' && r <= 'z':
		return 0x1D41A + (r - 'a')
	case r >= '0' && r <= '9':
		return 0x1D7CE + (r - '0')
	case r >= 'Α' && r <= 'Ω':
		return 0x1D6A8 + (r - 'Α')
	case r >= 'α' && r <= 'ω':
		return 0x1D6C2 + (r - 'α')
	}
	return r
}
