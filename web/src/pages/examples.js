import { PROJECTS, PLATFORM_NAMES } from "../projects.js";
import { escapeHtml } from "../markdown.js";
import { platformIcon, icon } from "./icons.js";

function asset(p, file) {
	return `examples/${p.slug}/${file}`;
}

function platformChips(list) {
	return `<div class="chips">${list.map((pl) => `<span class="chip">${platformIcon(pl)}${PLATFORM_NAMES[pl]}</span>`).join("")}</div>`;
}

export function renderExamples(view) {
	document.title = "Examples - Giode";
	view.innerHTML = `<section class="page-hero">
		<div class="container">
			<span class="eyebrow">Built with Giode</span>
			<h1>Real apps, <span class="grad">one Go codebase</span> each.</h1>
			<p class="lead">From a GPU video player to on-device AI on phones. Every one is open source: open its website to see it and download it.</p>
		</div>
	</section>
	<section class="container examples-grid">
		${PROJECTS.map((p, i) => `
		<article class="example-card reveal" style="--accent:${p.accent}; --i:${i}">
			<div class="example-top">
				<img class="app-icon" src="${asset(p, "icon.png")}" alt="" width="64" height="64" loading="lazy">
				<span class="category">${escapeHtml(p.category)}</span>
			</div>
			<h2><a class="card-link" href="${p.website}" target="_blank" rel="noopener">${escapeHtml(p.name)}</a></h2>
			<p>${escapeHtml(p.tagline)}</p>
			${platformChips(p.platforms)}
			<span class="more">Visit website ${icon("external", 14)}</span>
		</article>`).join("")}
		<a class="example-card add" href="#/docs/contributing">
			<div class="plus" aria-hidden="true">+</div>
			<h2>Your app here</h2>
			<p>Built something with Giode? Add it to this page with a pull request.</p>
			<span class="more">How to add it <span aria-hidden="true">→</span></span>
		</a>
	</section>`;
	observeReveal(view);
}

// openProject sends links to the old per-app pages (#/examples/<slug>,
// or a slug the app had before it was renamed) to the app's website.
export function openProject(view, slug) {
	const p = PROJECTS.find((x) => x.slug === slug || (x.aliases || []).includes(slug));
	if (p) {
		location.replace(p.website);
		return;
	}
	view.innerHTML = `<section class="notfound"><h1>Not found</h1><p>No example called <code>${escapeHtml(slug)}</code>.</p><a class="btn btn-primary" href="#/examples">All examples</a></section>`;
}

// observeReveal fades elements in as they scroll into view.
export function observeReveal(root) {
	const els = root.querySelectorAll(".reveal");
	if (!("IntersectionObserver" in window)) {
		els.forEach((e) => e.classList.add("in"));
		return;
	}
	const io = new IntersectionObserver((entries) => {
		for (const e of entries) {
			if (e.isIntersecting) {
				e.target.classList.add("in");
				io.unobserve(e.target);
			}
		}
	}, { rootMargin: "0px 0px -8% 0px" });
	els.forEach((e) => io.observe(e));
}
