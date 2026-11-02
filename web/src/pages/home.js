import snippets from "virtual:snippets";
import { highlight, escapeHtml } from "../markdown.js";
import { PROJECTS, PLATFORM_NAMES } from "../projects.js";
import { detectOS } from "../download.js";
import { icon, platformIcon } from "./icons.js";
import { observeReveal } from "./examples.js";
import { selectTab } from "../widgets.js";

const INSTALL = [
	{ label: "macOS / Linux", cmd: "curl -fsSL https://raw.githubusercontent.com/xmhrnx/giode-cli/main/install.sh | sh" },
	{ label: "Windows", cmd: "irm https://raw.githubusercontent.com/xmhrnx/giode-cli/main/install.ps1 | iex" },
	{ label: "Go", cmd: "go install github.com/xmhrnx/giode-cli/cmd/giode@latest" },
];

const FEATURES = [
	{ icon: "style", title: "Styled like CSS", text: "Padding, gaps, flexbox, borders, radii, shadows, gradients and hover states - 40+ properties on every element, with theme tokens for light and dark.", wide: true, demo: "props" },
	{ icon: "gpu", title: "Native and fast", text: "Drawn by Gio on the GPU. No webview, no JavaScript, no runtime to ship: a single binary that starts instantly." },
	{ icon: "platforms", title: "Every screen", text: "Windows, macOS, Linux, Android, iOS and the web from one codebase, with layouts that adapt to the width they get." },
	{ icon: "components", title: "Components included", text: "Buttons, inputs, menus, dialogs, sheets, tabs, toasts, tree views, virtual and lazy lists, title bars and more - all restylable." },
	{ icon: "state", title: "State that just works", text: "Plain Go variables, keyed per-item state, and observable State values any goroutine can write - the window redraws itself." },
	{ icon: "services", title: "Platform services", text: "Notifications, push, in-app purchases, tray icons, file pickers, sharing, secure storage, auto-update and crash reports." },
	{ icon: "cli", title: "A CLI that ships", text: "giode new, run and build: signed installers, MSIX, DMG, deb, rpm, AppImage, Flatpak, APK, AAB, IPA and a PWA.", wide: true, demo: "terminal" },
	{ icon: "ai", title: "Agent-ready", text: "A built-in MCP server lets AI agents see the UI tree, click, type and take screenshots - for testing and automation." },
	{ icon: "raw", title: "Raw Gio, anytime", text: "Every element is a Gio widget. Drop to paths, ops and gestures for custom drawing, draw video or 3D under the UI, or host Giode in your own window.", wide: true },
];

const SAMPLES = [
	{ key: "style", label: "Elements & styles", title: "Compose, then style", text: "Boxes, rows, stacks and text, arranged with flexbox and styled with properties you already know from CSS. Hover styles are one call away." },
	{ key: "state", label: "State", title: "State from anywhere", text: "A State value can be written from a goroutine - a network response, a timer - and the window redraws on its own. No channels, no locks, no invalidate calls." },
	{ key: "components", label: "Components", title: "Components with batteries", text: "Stateful components keep their own input, focus and animation. Create them once, lay them out every frame, chain their setters." },
	{ key: "responsive", label: "Responsive", title: "One view, every width", text: "Responsive switches layouts by the width a component is given, so the same code serves a phone and a 4K monitor." },
];

function code(src) {
	return `<pre><code class="lang-go">${highlight(src, "go")}</code></pre>`;
}

export function renderHome(view) {
	document.title = "Giode - native UI for Go";
	const { os } = detectOS();
	const installTab = os === "windows" ? 1 : 0;
	view.innerHTML = `
	<section class="hero">
		<div class="aurora" aria-hidden="true"><i></i><i></i><i></i></div>
		<div class="grid-bg" aria-hidden="true"></div>
		<div class="container hero-inner">
			<div class="hero-copy">
				<a class="alpha-pill" href="#/docs/faq#is-it-ready-for-production">
					<b>Alpha</b> Giode is in alpha - expect breaking changes <span aria-hidden="true">→</span>
				</a>
				<h1>Native apps<br>in <span class="grad">pure Go</span>,<br>for every screen.</h1>
				<p class="lead">Giode brings the ease of retained-mode UI to <a href="https://gioui.org" target="_blank" rel="noopener">Gio</a>'s immediate-mode speed. Compose elements, style them like CSS, and ship one codebase to desktop, mobile and the web - no webview, no JavaScript.</p>
				<div class="cta">
					<a class="btn btn-primary btn-lg" href="#/docs/getting-started">Get started ${icon("arrow", 18)}</a>
					<a class="btn btn-ghost btn-lg" href="#/examples">See what's built with it</a>
				</div>
				<div class="install" role="group" aria-label="Install the giode CLI">
					<div class="tab-head">${INSTALL.map((t, i) => `<button class="tab${i === installTab ? " active" : ""}" data-group="install" data-tab="${i}">${t.label}</button>`).join("")}</div>
					${INSTALL.map((t, i) => `<div class="tab-panel install-line${i === installTab ? " active" : ""}" data-group="install" data-tab="${i}">
						<span class="prompt">$</span><code>${escapeHtml(t.cmd)}</code>
						<button class="copy-btn" data-copy="${escapeHtml(t.cmd)}" aria-label="Copy the command">${icon("copy", 16)}<span class="copy-label">Copy</span></button>
					</div>`).join("")}
				</div>
			</div>
			<div class="hero-demo">
				<div class="code-window">
					<div class="window-bar"><i></i><i></i><i></i><span>main.go</span></div>
					${code(snippets.hero)}
				</div>
				<div class="app-preview" data-mode="dark">
					<div class="window-bar"><i></i><i></i><i></i><span>Counter</span>
						<button class="preview-mode" aria-label="Switch the preview's theme">${icon("spark", 14)}</button></div>
					<div class="preview-body">
						<div class="preview-count" aria-live="polite">0</div>
						<button class="preview-btn">Add one</button>
						<div class="preview-hint">It's live - try it</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<section class="platforms reveal">
		<div class="container">
			<p class="platforms-title">One codebase. Six platforms.</p>
			<div class="platform-row">${Object.keys(PLATFORM_NAMES).map((p) => `<div class="platform">${platformIcon(p, 28)}<span>${PLATFORM_NAMES[p]}</span></div>`).join("")}</div>
		</div>
	</section>

	<section class="container section">
		<div class="section-head reveal">
			<span class="eyebrow">Why Giode</span>
			<h2>Everything an app needs,<br><span class="grad">nothing it doesn't.</span></h2>
			<p class="lead">Gio gives you a fast, portable canvas. Giode turns it into an app framework without taking the canvas away.</p>
		</div>
		<div class="bento">
			${FEATURES.map((f, i) => `<div class="feature reveal${f.wide ? " wide" : ""}" style="--i:${i % 3}">
				<div class="feature-icon">${icon(f.icon)}</div>
				<h3>${f.title}</h3>
				<p>${f.text}</p>
				${f.demo === "props" ? propsDemo() : ""}
				${f.demo === "terminal" ? miniTerminal() : ""}
			</div>`).join("")}
		</div>
	</section>

	<section class="container section samples">
		<div class="section-head reveal">
			<span class="eyebrow">Write it like you think</span>
			<h2>Plain Go, top to bottom.</h2>
			<p class="lead">No templates, no code generation, no framework rules. Functions return elements; structs remember things. <a href="#/docs/recipes">See the recipes →</a></p>
		</div>
		<div class="sample-tabs reveal">
			<div class="sample-nav" role="tablist">${SAMPLES.map((s, i) => `<button class="tab sample-tab${i === 0 ? " active" : ""}" data-group="samples" data-tab="${i}" role="tab">${s.label}</button>`).join("")}</div>
			${SAMPLES.map((s, i) => `<div class="tab-panel sample${i === 0 ? " active" : ""}" data-group="samples" data-tab="${i}">
				<div class="sample-text"><h3>${s.title}</h3><p>${s.text}</p></div>
				<div class="code-window">${code(snippets[s.key])}</div>
			</div>`).join("")}
		</div>
	</section>

	<section class="section cli-section">
		<div class="container cli-inner">
			<div class="reveal">
				<span class="eyebrow">The giode CLI</span>
				<h2>From <code>giode new</code> to the stores.</h2>
				<p class="lead">Pick your platforms from a checklist, answer a few questions, and get a project set up for exactly those. Then run it on your phone with one command, and build signed installers and store bundles with another.</p>
				<ul class="checks">
					<li>Runs on your desktop, a phone, a simulator or the browser</li>
					<li>Installers, MSIX, DMG, deb, rpm, AppImage, Flatpak, APK, AAB, IPA</li>
					<li>Signing, notarization, native libraries and a CI workflow</li>
				</ul>
				<a class="btn btn-ghost" href="#/docs/cli/index">Read about the CLI ${icon("arrow", 16)}</a>
			</div>
			<div class="terminal reveal" aria-label="A giode new session">
				<div class="window-bar"><i></i><i></i><i></i><span>Terminal</span></div>
				<pre class="term-body"></pre>
			</div>
		</div>
	</section>

	<section class="container section">
		<div class="section-head reveal">
			<span class="eyebrow">Showcase</span>
			<h2>Built with Giode.</h2>
			<p class="lead">Open-source apps you can download today - and read the code of.</p>
		</div>
		<div class="showcase">
			${PROJECTS.map((p, i) => `<a class="show-card reveal" href="#/examples/${p.slug}" style="--accent:${p.accent}; --i:${i}">
				<img src="examples/${p.slug}/icon.png" alt="" width="56" height="56" loading="lazy">
				<div><b>${escapeHtml(p.name)}</b><span>${escapeHtml(p.tagline)}</span></div>
				<div class="show-plats">${p.platforms.map((pl) => platformIcon(pl, 14)).join("")}</div>
			</a>`).join("")}
		</div>
		<div class="center"><a class="btn btn-ghost" href="#/examples">All examples ${icon("arrow", 16)}</a></div>
	</section>

	<section class="container section">
		<div class="final-cta reveal">
			<div class="aurora small" aria-hidden="true"><i></i><i></i></div>
			<h2>Build your first Giode app<br>in five minutes.</h2>
			<p>Install the CLI, run <code>giode new</code>, pick your platforms, and you're running.</p>
			<div class="cta">
				<a class="btn btn-primary btn-lg" href="#/docs/getting-started">Get started ${icon("arrow", 18)}</a>
				<a class="btn btn-ghost btn-lg" href="https://github.com/xmhrnx/giode" target="_blank" rel="noopener">Star on GitHub</a>
			</div>
			<p class="alpha-note">Giode is in alpha: it's used in real apps, but the API will change before 1.0.</p>
		</div>
	</section>`;

	wirePreview(view);
	observeReveal(view);
	startTerminal(view.querySelector(".term-body"));
	selectTab("install", installTab, view.querySelector(".install"));
}

function propsDemo() {
	const props = [["Padding", "UniformInset(20)"], ["Gap", "12"], ["BorderRadius", "16"], ["BoxShadow", "{Y: 8, Blur: 24}"], ["Background", "\"theme.surface\""], ["Justify", "JustifyBetween"]];
	return `<div class="props-demo">${props.map(([k, v]) => `<span><b>${k}</b>: ${escapeHtml(v)}</span>`).join("")}</div>`;
}

function miniTerminal() {
	return `<div class="mini-term"><span class="t-dim">$</span> giode build -release android aab windows-installer dmg<br>
		<span class="t-ok">✔</span> dist/android/Notes.apk<br><span class="t-ok">✔</span> dist/android/Notes.aab<br>
		<span class="t-ok">✔</span> dist/windows/Notes-1.0.0-setup.exe<br><span class="t-ok">✔</span> dist/macos/Notes-1.0.0.dmg</div>`;
}

function wirePreview(view) {
	const preview = view.querySelector(".app-preview");
	const count = preview.querySelector(".preview-count");
	let n = 0;
	preview.querySelector(".preview-btn").addEventListener("click", () => {
		n++;
		count.textContent = String(n);
		count.classList.remove("bump");
		void count.offsetWidth;
		count.classList.add("bump");
		preview.querySelector(".preview-hint").textContent = n === 1 ? "That's a Giode button, in HTML" : `${n} clicks`;
	});
	preview.querySelector(".preview-mode").addEventListener("click", () => {
		preview.dataset.mode = preview.dataset.mode === "dark" ? "light" : "dark";
	});
}

// The terminal types out a giode new session when it scrolls into view.
const SESSION = [
	["$ ", "giode new", "cmd"],
	["", "? Which platforms is the app for?", "q"],
	["", "  [x] Windows   [x] macOS   [ ] Linux", "opt"],
	["", "❯ [x] Android   [x] iOS     [ ] Web", "opt"],
	["", "? App name (Notes) ", "q"],
	["", "? App ID (com.example.notes) com.acme.notes", "q"],
	["", "? Version (0.1.0) ", "q"],
	["", "✔ Created notes", "ok"],
	["$ ", "cd notes && giode run android", "cmd"],
	["", "Installing on Pixel 8… launched com.acme.notes", "dim"],
];

function startTerminal(el) {
	if (!el) return;
	const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
	const full = SESSION.map(([p, t, c]) => `<span class="t-${c}">${escapeHtml(p + t)}</span>`).join("\n");
	if (reduce || !("IntersectionObserver" in window)) {
		el.innerHTML = full;
		return;
	}
	let started = false;
	const io = new IntersectionObserver((entries) => {
		if (!entries.some((e) => e.isIntersecting) || started) return;
		started = true;
		io.disconnect();
		let line = 0;
		let ch = 0;
		const done = [];
		const tick = () => {
			if (!el.isConnected) return;
			if (line >= SESSION.length) {
				el.innerHTML = done.join("\n") + '\n<span class="t-cmd">$ </span><span class="cursor"></span>';
				return;
			}
			const [p, t, c] = SESSION[line];
			const typing = c === "cmd";
			if (typing && ch < t.length) {
				ch++;
				el.innerHTML = done.concat(`<span class="t-${c}">${escapeHtml(p + t.slice(0, ch))}</span><span class="cursor"></span>`).join("\n");
				setTimeout(tick, 45 + Math.random() * 40);
				return;
			}
			done.push(`<span class="t-${c}">${escapeHtml(p + t)}</span>`);
			el.innerHTML = done.join("\n");
			line++;
			ch = 0;
			setTimeout(tick, typing ? 380 : 260);
		};
		tick();
	}, { threshold: 0.4 });
	io.observe(el);
}
