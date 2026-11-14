package formula

import (
	"strings"
	"unicode"
)

// The tree a formula parses into.
type (
	node interface{}
	// atom is a symbol or a run of them (digits) set in the math font.
	atom struct {
		s   string
		cls class
	}
	// textRun is \text: upright text in the text font.
	textRun struct{ s string }
	group   struct{ body []node }
	frac    struct {
		num, den []node
		style    int // 0 by context, 1 \dfrac, 2 \tfrac
	}
	radical struct{ body, index []node }
	scripts struct {
		base     node
		sup, sub []node
	}
	bigOp struct {
		s      string
		limits bool
	}
	function struct {
		name   string
		limits bool
	}
	accent struct {
		kind string
		body []node
	}
	space    struct{ em float32 }
	delimits struct {
		left, right string
		body        []node
	}
	sizedDelim struct {
		s     string
		scale float32
		cls   class
	}
	lineBreak struct{}
	alignMark struct{}
)

// font is how letters are set.
type font uint8

const (
	fontItalic font = iota
	fontUpright
	fontBold
)

type parser struct {
	src  []rune
	i    int
	font font
}

// parse reads a formula: the subset of LaTeX math the package sets.
func parse(src string) []node {
	p := &parser{src: []rune(src)}
	return p.list(false)
}

func (p *parser) peek() rune {
	if p.i < len(p.src) {
		return p.src[p.i]
	}
	return 0
}

func (p *parser) skipSpace() {
	for p.i < len(p.src) && unicode.IsSpace(p.src[p.i]) {
		p.i++
	}
}

// list parses until the end, a closing brace (when inGroup, consumed) or
// \right (left for the caller).
func (p *parser) list(inGroup bool) []node {
	var out []node
	for {
		p.skipSpace()
		if p.i >= len(p.src) {
			return out
		}
		c := p.src[p.i]
		switch c {
		case '}':
			if inGroup {
				p.i++
				return out
			}
			p.i++ // a stray brace
			continue
		case '{':
			p.i++
			out = append(out, group{body: p.list(true)})
		case '^', '_':
			p.i++
			arg := p.arg()
			out = attach(out, c == '^', arg)
		case '\'':
			// Primes follow the base at full size: the glyphs sit high.
			n := 0
			for p.i < len(p.src) && p.src[p.i] == '\'' {
				p.i++
				n++
			}
			primes := strings.Repeat("‴", n/3) + []string{"", "′", "″"}[n%3]
			out = append(out, atom{primes, clsOrd})
		case '&':
			p.i++
			out = append(out, alignMark{})
		case '\\':
			if p.at(`\right`) {
				return out
			}
			if n, ok := p.command(); ok {
				out = append(out, n...)
			}
		default:
			out = append(out, p.char())
		}
	}
}

// at reports whether the source continues with cmd, a whole command.
func (p *parser) at(cmd string) bool {
	r := []rune(cmd)
	if p.i+len(r) > len(p.src) || string(p.src[p.i:p.i+len(r)]) != cmd {
		return false
	}
	next := p.i + len(r)
	return next >= len(p.src) || !isLetter(p.src[next])
}

func isLetter(r rune) bool { return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' }

// attach gives the last node of out a superscript or subscript.
func attach(out []node, sup bool, arg []node) []node {
	var s scripts
	if n := len(out); n > 0 {
		if prev, ok := out[n-1].(scripts); ok && (sup && prev.sup == nil || !sup && prev.sub == nil) {
			s = prev
			out = out[:n-1]
		} else {
			s.base = out[n-1]
			out = out[:n-1]
		}
	}
	if s.base == nil {
		s.base = group{}
	}
	if sup {
		s.sup = arg
	} else {
		s.sub = arg
	}
	return append(out, s)
}

// arg parses a command's or script's argument: a group, a command or a
// character.
func (p *parser) arg() []node {
	p.skipSpace()
	if p.i >= len(p.src) {
		return nil
	}
	switch c := p.src[p.i]; c {
	case '{':
		p.i++
		return p.list(true)
	case '\\':
		n, _ := p.command()
		return n
	default:
		return []node{p.char()}
	}
}

// rawGroup returns a braced argument's text as written.
func (p *parser) rawGroup() string {
	p.skipSpace()
	if p.peek() != '{' {
		if p.i < len(p.src) {
			p.i++
			return string(p.src[p.i-1])
		}
		return ""
	}
	p.i++
	depth, start := 1, p.i
	for p.i < len(p.src) {
		switch p.src[p.i] {
		case '\\':
			p.i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				s := string(p.src[start:p.i])
				p.i++
				return s
			}
		}
		p.i++
	}
	return string(p.src[start:])
}

// char parses one character, or a run of digits.
func (p *parser) char() node {
	c := p.src[p.i]
	p.i++
	if c >= '0' && c <= '9' {
		start := p.i - 1
		for p.i < len(p.src) && (p.src[p.i] >= '0' && p.src[p.i] <= '9' || p.src[p.i] == '.' && p.i+1 < len(p.src) && p.src[p.i+1] >= '0' && p.src[p.i+1] <= '9') {
			p.i++
		}
		s := string(p.src[start:p.i])
		if p.font == fontBold {
			s = strings.Map(bold, s)
		}
		return atom{s, clsOrd}
	}
	switch c {
	case '-':
		return atom{"−", clsBin}
	case '*':
		return atom{"∗", clsBin}
	}
	return atom{p.letter(c), charClass(c)}
}

// letter sets a letter in the current font.
func (p *parser) letter(c rune) string {
	switch p.font {
	case fontUpright:
		return string(c)
	case fontBold:
		return string(bold(c))
	}
	if isLetter(c) || c >= 'α' && c <= 'ω' || strings.ContainsRune("ϵϑϕϱϖ∂", c) {
		return string(italic(c))
	}
	return string(c)
}

// command parses a backslash command.
func (p *parser) command() ([]node, bool) {
	p.i++ // the backslash
	if p.i >= len(p.src) {
		return nil, false
	}
	var name string
	if isLetter(p.src[p.i]) {
		start := p.i
		for p.i < len(p.src) && isLetter(p.src[p.i]) {
			p.i++
		}
		name = string(p.src[start:p.i])
	} else {
		name = string(p.src[p.i])
		p.i++
	}
	switch name {
	case "\\", "cr", "newline":
		return []node{lineBreak{}}, true
	case "frac", "dfrac", "tfrac", "cfrac":
		style := map[string]int{"dfrac": 1, "tfrac": 2, "cfrac": 1}[name]
		num := p.arg()
		den := p.arg()
		return []node{frac{num: num, den: den, style: style}}, true
	case "binom":
		num := p.arg()
		den := p.arg()
		return []node{delimits{left: "(", right: ")", body: []node{frac{num: num, den: den, style: -1}}}}, true
	case "sqrt":
		var index []node
		p.skipSpace()
		if p.peek() == '[' {
			p.i++
			start := p.i
			for p.i < len(p.src) && p.src[p.i] != ']' {
				p.i++
			}
			index = parseIn(string(p.src[start:p.i]), p.font)
			if p.i < len(p.src) {
				p.i++
			}
		}
		return []node{radical{body: p.arg(), index: index}}, true
	case "text", "textrm", "textnormal", "mbox", "textup":
		return []node{textRun{s: p.rawGroup()}}, true
	case "mathrm", "operatorname", "mathup", "rm":
		return p.inFont(fontUpright, name == "operatorname"), true
	case "mathbf", "boldsymbol", "bm", "textbf":
		return p.inFont(fontBold, false), true
	case "mathit", "textit":
		return p.inFont(fontItalic, false), true
	case "left":
		left := p.delimiter()
		body := p.list(false)
		right := "."
		if p.at(`\right`) {
			p.i += len(`\right`)
			right = p.delimiter()
		}
		return []node{delimits{left: left, right: right, body: body}}, true
	case "right":
		p.delimiter()
		return nil, false
	case "big", "Big", "bigg", "Bigg", "bigl", "Bigl", "biggl", "Biggl", "bigr", "Bigr", "biggr", "Biggr", "bigm", "Bigm":
		// \big to \Bigg: 1.2, 1.8, 2.4 and 3 em tall, as TeX's.
		scale := map[byte]float32{'b': 1.2, 'B': 1.8}[name[0]]
		if strings.HasPrefix(strings.ToLower(name), "bigg") {
			scale += 1.2
		}
		d := p.delimiter()
		cls := clsOrd
		switch {
		case strings.HasSuffix(name, "l"):
			cls = clsOpen
		case strings.HasSuffix(name, "r"):
			cls = clsClose
		case strings.HasSuffix(name, "m"):
			cls = clsRel
		}
		return []node{sizedDelim{s: d, scale: scale, cls: cls}}, true
	case "vec", "hat", "widehat", "bar", "overline", "dot", "ddot", "tilde", "widetilde", "overrightarrow", "underline", "check", "breve", "acute", "grave":
		return []node{accent{kind: name, body: p.arg()}}, true
	case "begin", "end":
		p.rawGroup()
		return nil, false
	case "displaystyle", "textstyle", "nonumber", "notag", "label":
		if name == "label" {
			p.rawGroup()
		}
		return nil, false
	case "limits", "nolimits":
		return nil, false
	case "not":
		n, _ := p.commandOrChar()
		if a, ok := n.(atom); ok {
			a.s += "̸"
			return []node{a}, true
		}
		return []node{n}, true
	}
	if em, ok := spaces[name]; ok {
		return []node{space{em}}, true
	}
	if op, ok := bigOps[name]; ok {
		return p.limits(bigOp{s: op.s, limits: op.limits}), true
	}
	if limits, ok := functions[name]; ok {
		return []node{function{name: name, limits: limits}}, true
	}
	if s, ok := symbols[name]; ok {
		str := s.s
		if p.font == fontItalic && len(name) > 1 && unicode.IsLower(rune(name[0])) {
			str = string(italic([]rune(s.s)[0]))
		}
		if p.font == fontBold {
			str = strings.Map(bold, str)
		}
		return []node{atom{str, s.cls}}, true
	}
	// Unknown: shown as written, so the mistake is visible.
	return []node{textRun{s: `\` + name}}, true
}

// limits reads \limits or \nolimits after a big operator.
func (p *parser) limits(op bigOp) []node {
	p.skipSpace()
	switch {
	case p.at(`\limits`):
		p.i += len(`\limits`)
		op.limits = true
	case p.at(`\nolimits`):
		p.i += len(`\nolimits`)
		op.limits = false
	}
	return []node{op}
}

func (p *parser) commandOrChar() (node, bool) {
	p.skipSpace()
	if p.peek() == '\\' {
		n, ok := p.command()
		if len(n) > 0 {
			return n[0], ok
		}
		return group{}, false
	}
	if p.i < len(p.src) {
		return p.char(), true
	}
	return group{}, false
}

// inFont parses an argument in another font.
func (p *parser) inFont(f font, operator bool) []node {
	old := p.font
	p.font = f
	body := p.arg()
	p.font = old
	if operator {
		var b strings.Builder
		for _, n := range body {
			if a, ok := n.(atom); ok {
				b.WriteString(a.s)
			}
		}
		return []node{function{name: b.String()}}
	}
	return []node{group{body: body}}
}

// delimiter reads the delimiter after \left, \right or \big.
func (p *parser) delimiter() string {
	p.skipSpace()
	if p.i >= len(p.src) {
		return "."
	}
	c := p.src[p.i]
	if c != '\\' {
		p.i++
		return string(c)
	}
	p.i++
	start := p.i
	if p.i < len(p.src) && !isLetter(p.src[p.i]) {
		p.i++
	} else {
		for p.i < len(p.src) && isLetter(p.src[p.i]) {
			p.i++
		}
	}
	name := string(p.src[start:p.i])
	switch name {
	case "{", "lbrace":
		return "{"
	case "}", "rbrace":
		return "}"
	case "|", "Vert":
		return "‖"
	case "vert":
		return "|"
	}
	if s, ok := symbols[name]; ok {
		return s.s
	}
	return "."
}

// parseIn parses src with letters in font f.
func parseIn(src string, f font) []node {
	p := &parser{src: []rune(src), font: f}
	return p.list(false)
}
