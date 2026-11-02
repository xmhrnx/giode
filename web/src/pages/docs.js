import { NAV, PAGES } from "../nav.js";
import { render, escapeHtml } from "../markdown.js";
import { detectOS } from "../download.js";
import { selectTab } from "../widgets.js";

// The docs: Markdown from ../docs, with the navigation on the left, the
// page's sections on the right, and previous/next links.

const cache = new Map();

// files maps a page to the Markdown file it came from.
const files = new Map();

async function fetchDoc(path) {
	if (cache.has(path)) return cache.get(path);
	let text = null;
	for (const c of [`${path}.md`, `${path}/README.md`, `${path}/index.md`]) {
		try {
			const r = await fetch("docs/" + c);
			// Dev servers answer unknown paths with index.html.
			if (r.ok && !(r.headers.get("content-type") || "").includes("text/html")) {
				text = await r.text();
				files.set(path, c);
				break;
			}
		} catch (e) {
			// next candidate
		}
	}
	cache.set(path, text);
	return text;
}

export async function hasDoc(path) {
	return PAGES.some((p) => p.path === path) || (await fetchDoc(path)) !== null;
}

// resolve turns a link in a doc into a site link: other docs become
// #/docs/..., anchors stay on the page, the rest opens in a new tab.
function resolve(href, current) {
	if (!href) return null;
	if (href.startsWith("#/")) return { href };
	if (href.startsWith("#")) return { href: `#/docs/${current}${href}` };
	if (/^[a-z]+:/i.test(href)) return { href, external: true };
	let [path, frag] = href.split("#");
	path = path.replace(/\.md$/, "").replace(/\/README$/, "").replace(/\/$/, "");
	const dir = current.includes("/") ? current.slice(0, current.lastIndexOf("/")) : "";
	const parts = (dir ? dir.split("/") : []);
	for (const seg of path.split("/")) {
		if (seg === "..") parts.pop();
		else if (seg && seg !== ".") parts.push(seg);
	}
	let target = parts.join("/") || "index";
	if (path === "") target = current;
	return { href: `#/docs/${target}${frag ? "#" + frag : ""}` };
}

function sidebar(current) {
	return NAV.map((s) => `<div class="side-section"><div class="side-title">${s.title}</div>` +
		s.items.map((it) => `<a class="side-link${it.path === current ? " active" : ""}" href="#/docs/${it.path}">${it.label}</a>`).join("") +
		`</div>`).join("");
}

function prevNext(current) {
	const i = PAGES.findIndex((p) => p.path === current);
	if (i < 0) return "";
	const prev = PAGES[i - 1];
	const next = PAGES[i + 1];
	return `<nav class="prevnext" aria-label="Previous and next pages">` +
		(prev ? `<a class="pn prev" href="#/docs/${prev.path}"><span>Previous</span>${prev.label}</a>` : "<span></span>") +
		(next ? `<a class="pn next" href="#/docs/${next.path}"><span>Next</span>${next.label}</a>` : "<span></span>") +
		`</nav>`;
}

let searchIndex = null;

async function search(q) {
	if (!searchIndex) {
		searchIndex = Promise.all(PAGES.map(async (p) => ({ ...p, text: (await fetchDoc(p.path)) || "" })));
	}
	const docs = await searchIndex;
	const terms = q.toLowerCase().split(/\s+/).filter(Boolean);
	const hits = [];
	for (const d of docs) {
		const hay = d.text.toLowerCase();
		let score = 0;
		for (const t of terms) {
			if (d.label.toLowerCase().includes(t)) score += 5;
			else if (hay.includes(t)) score += 1 + Math.min(3, hay.split(t).length - 1) / 3;
			else {
				score = 0;
				break;
			}
		}
		if (score > 0) {
			const at = hay.indexOf(terms[0]);
			const snip = at >= 0 ? d.text.slice(Math.max(0, at - 40), at + 90).replace(/[#`*|>]/g, "").replace(/\s+/g, " ") : "";
			hits.push({ ...d, score, snip });
		}
	}
	return hits.sort((a, b) => b.score - a.score).slice(0, 8);
}

function wireSearch(root) {
	const input = root.querySelector(".doc-search input");
	const results = root.querySelector(".doc-search .results");
	let timer;
	input.addEventListener("input", () => {
		clearTimeout(timer);
		const q = input.value.trim();
		if (!q) {
			results.classList.remove("open");
			return;
		}
		timer = setTimeout(async () => {
			const hits = await search(q);
			results.innerHTML = hits.length
				? hits.map((h) => `<a href="#/docs/${h.path}"><b>${h.label}</b><small>${h.section}</small>${h.snip ? `<span>…${escapeHtml(h.snip)}…</span>` : ""}</a>`).join("")
				: `<div class="empty">Nothing found for “${escapeHtml(q)}”.</div>`;
			results.classList.add("open");
		}, 120);
	});
	input.addEventListener("keydown", (e) => {
		if (e.key === "Escape") {
			input.value = "";
			results.classList.remove("open");
			input.blur();
		} else if (e.key === "Enter") {
			const first = results.querySelector("a");
			if (first) location.hash = first.getAttribute("href");
		}
	});
	document.addEventListener("click", (e) => {
		if (!e.target.closest(".doc-search")) results.classList.remove("open");
	});
	document.onkeydown = (e) => {
		if (e.key === "/" && !["INPUT", "TEXTAREA"].includes(document.activeElement.tagName) && document.body.dataset.page === "docs") {
			e.preventDefault();
			input.focus();
		}
	};
}

// spy highlights the section being read in the page's table of contents.
let spy;

function wireToc(root) {
	if (spy) spy.disconnect();
	const links = new Map([...root.querySelectorAll(".toc a")].map((a) => [a.dataset.id, a]));
	if (!links.size) return;
	spy = new IntersectionObserver((entries) => {
		for (const e of entries) {
			if (e.isIntersecting) {
				links.forEach((a) => a.classList.remove("active"));
				const a = links.get(e.target.id);
				if (a) a.classList.add("active");
			}
		}
	}, { rootMargin: "-80px 0px -70% 0px" });
	root.querySelectorAll(".prose h2[id], .prose h3[id]").forEach((h) => spy.observe(h));
}

// preferOSTabs shows the tab for the visitor's system first: "Windows",
// "macOS / Linux".
function preferOSTabs(root) {
	const { os } = detectOS();
	const names = { windows: /windows/i, macos: /mac/i, linux: /linux|debian|ubuntu/i }[os];
	if (!names) return;
	root.querySelectorAll(".tabs").forEach((t) => {
		const btn = [...t.querySelectorAll(".tab")].find((b) => names.test(b.textContent));
		if (btn) selectTab(btn.dataset.group, btn.dataset.tab, t);
	});
}

export async function renderDocs(view, path) {
	path = path.replace(/\/$/, "");
	if (!view.querySelector(".docs-layout")) {
		view.innerHTML = `<div class="docs-layout">
			<aside class="docs-side">
				<div class="doc-search"><input type="search" placeholder="Search docs" aria-label="Search the docs" autocomplete="off"><kbd>/</kbd><div class="results"></div></div>
				<nav class="side-nav" aria-label="Documentation"></nav>
			</aside>
			<article class="docs-main"></article>
			<aside class="docs-toc"></aside>
		</div>`;
		wireSearch(view);
	}
	view.querySelector(".side-nav").innerHTML = sidebar(path);
	const main = view.querySelector(".docs-main");
	const tocEl = view.querySelector(".docs-toc");
	main.innerHTML = `<div class="doc-loading"><span></span><span></span><span></span></div>`;
	const text = await fetchDoc(path);
	if (text === null) {
		main.innerHTML = `<div class="prose"><h1>Page not found</h1><p>There's no page <code>${escapeHtml(path)}</code>. Try the search, or start at <a href="#/docs/getting-started">Getting started</a>.</p></div>`;
		tocEl.innerHTML = "";
		document.title = "Not found - Giode docs";
		return;
	}
	const { html, toc } = render(text);
	const page = PAGES.find((p) => p.path === path);
	const crumbs = page ? `<div class="crumbs"><a href="#/docs/index">Docs</a><span>/</span>${page.section}</div>` : "";
	const file = files.get(path) || path + ".md";
	main.innerHTML = `${crumbs}<div class="prose">${html}</div>` +
		`<div class="doc-foot"><a href="https://github.com/xmhrnx/giode/edit/main/docs/${file}" target="_blank" rel="noopener">Edit this page on GitHub</a></div>` +
		prevNext(path);
	main.querySelectorAll(".prose a[href]").forEach((a) => {
		const r = resolve(a.getAttribute("href"), path);
		if (!r) return;
		a.setAttribute("href", r.href);
		if (r.external) {
			a.target = "_blank";
			a.rel = "noopener";
		}
	});
	tocEl.innerHTML = toc.length > 1
		? `<div class="toc"><div class="toc-title">On this page</div>${toc.map((h) => `<a class="l${h.level}" data-id="${h.id}" href="#/docs/${path}#${h.id}">${escapeHtml(h.text)}</a>`).join("")}</div>`
		: "";
	preferOSTabs(main);
	wireToc(view);
	const h1 = text.match(/^#\s+(.*)$/m);
	document.title = (h1 ? h1[1].replace(/`/g, "") : "Docs") + " - Giode";
}
