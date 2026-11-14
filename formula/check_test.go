package formula

import "testing"

func TestCheck(t *testing.T) {
	for _, src := range []string{
		`x^2 + y^2 = r^2`,
		`\frac{-b \pm \sqrt{b^2 - 4ac}}{2a}`,
		`\int_0^\infty e^{-x^2}\,\mathrm{d}x = \frac{\sqrt{\pi}}{2}`,
		`\left( \sum_{i=1}^n a_i \right)^2 \le n \sum_{i=1}^n a_i^2`,
		`\begin{pmatrix} a & b \\ c & d \end{pmatrix}`,
		`f(x) = \begin{cases} x & x \ge 0 \\ -x & \text{otherwise} \end{cases}`,
		`\begin{aligned} a &= b \\ &= c \end{aligned}`,
		`\vec{F} = m\vec{a}, \quad \nabla \cdot \vec{E} = \frac{\rho}{\varepsilon_0}`,
		`\mathcal{L} = \underbrace{a+b}_{\text{sum}}`,
		`\textcolor{red}{x} \neq \operatorname{tr}(A)`,
		`M_\odot \approx 2 \times 10^{30}\,\mathrm{kg}`,
		`\left\{ x \in \mathbb{R} \;\middle|\; x > 0 \right.`,
	} {
		if err := Check(src); err != nil {
			t.Errorf("%s: %v", src, err)
		}
	}
	for _, src := range []string{`\frac{a}{`, `x^`, `\left( x`, `\foo x`, `a } b`, `\begin{matrix} a`, `\begin{matrix} a \end{pmatrix}`, `x \right)`} {
		if Check(src) == nil {
			t.Errorf("%s: no error", src)
		}
	}
}
