import { PROJECTS, PLATFORM_NAMES } from "../projects.js";
import { escapeHtml } from "../markdown.js";
import { detectOS, latestRelease, pick, formatSize, fileKind } from "../download.js";
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
			<p class="lead">From a GPU video player to on-device AI on phones. Every one is open source: read the code, then download it for your system.</p>
		</div>
	</section>
	<section class="container examples-grid">
		${PROJECTS.map((p, i) => `
		<article class="example-card reveal" style="--accent:${p.accent}; --i:${i}">
			<div class="example-top">
				<img class="app-icon" src="${asset(p, "icon.png")}" alt="" width="64" height="64" loading="lazy">
				<span class="category">${escapeHtml(p.category)}</span>
			</div>
			<h2><a class="card-link" href="#/examples/${p.slug}">${escapeHtml(p.name)}</a></h2>
			<p>${escapeHtml(p.tagline)}</p>
			${platformChips(p.platforms)}
			<div class="card-foot">
				<span class="more">Details and download <span aria-hidden="true">→</span></span>
				${p.website ? `<a class="site-link" href="${p.website}" target="_blank" rel="noopener" aria-label="${escapeHtml(p.name)} website">${icon("globe", 15)}Website</a>` : ""}
			</div>
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

export function renderProject(view, slug) {
	const p = PROJECTS.find((x) => x.slug === slug);
	if (!p) {
		// An app that was renamed keeps its old links working.
		const renamed = PROJECTS.find((x) => (x.aliases || []).includes(slug));
		if (renamed) {
			location.replace(`#/examples/${renamed.slug}`);
			return;
		}
		view.innerHTML = `<section class="notfound"><h1>Not found</h1><p>No example called <code>${escapeHtml(slug)}</code>.</p><a class="btn btn-primary" href="#/examples">All examples</a></section>`;
		return;
	}
	document.title = `${p.name} - Giode examples`;
	const repoURL = `https://github.com/${p.repo}`;
	const phones = p.screenshots.filter((s) => s.phone);
	const wide = p.screenshots.filter((s) => !s.phone);
	view.innerHTML = `<section class="project-hero" style="--accent:${p.accent}">
		<div class="container">
			<a class="back" href="#/examples">← All examples</a>
			<div class="project-head">
				<img class="app-icon big" src="${asset(p, "icon.png")}" alt="${escapeHtml(p.name)} icon" width="112" height="112">
				<div>
					<span class="eyebrow">${escapeHtml(p.category)}</span>
					<h1>${escapeHtml(p.name)}</h1>
					<p class="lead">${escapeHtml(p.tagline)}</p>
					${platformChips(p.platforms)}
					<div class="project-links">
						${p.website ? `<a class="btn btn-ghost" href="${p.website}" target="_blank" rel="noopener">${icon("globe", 18)}Website${icon("external", 14)}</a>` : ""}
						<a class="btn btn-ghost" href="${repoURL}" target="_blank" rel="noopener">${icon("github", 18)}GitHub</a>
					</div>
				</div>
			</div>
			<div class="download-box" id="download" aria-live="polite">
				<div class="dl-loading">Finding the right download for your system…</div>
			</div>
		</div>
	</section>
	<section class="container project-body">
		<div class="project-about">
			<h2>About</h2>
			<p>${escapeHtml(p.description)}</p>
			<h3>Highlights</h3>
			<ul class="checks">${p.highlights.map((h) => `<li>${escapeHtml(h)}</li>`).join("")}</ul>
		</div>
		<aside class="project-side">
			<div class="side-card">
				<h3>Made with Giode</h3>
				<ul>${p.giode.map((g) => `<li>${escapeHtml(g)}</li>`).join("")}</ul>
			</div>
			${p.website ? `<div class="side-card">
				<h3>Website</h3>
				<a class="repo-link" href="${p.website}" target="_blank" rel="noopener">${escapeHtml(p.website.replace(/^https:\/\//, "").replace(/\/$/, ""))}</a>
				<p class="muted">${escapeHtml(p.name)}'s own website.</p>
			</div>` : ""}
			<div class="side-card">
				<h3>Source</h3>
				<a class="repo-link" href="${repoURL}" target="_blank" rel="noopener">${escapeHtml(p.repo)}</a>
				<p class="muted">Open source on GitHub: read how it's built, file issues, or build it yourself.</p>
			</div>
		</aside>
	</section>
	${p.screenshots.length ? `<section class="container shots">
		<h2>Screenshots</h2>
		${phones.length ? `<div class="shots-phones">${phones.map((s) => `<figure class="phone-frame"><img src="${asset(p, s.src)}" alt="${escapeHtml(s.alt)}" loading="lazy"><figcaption>${escapeHtml(s.alt)}</figcaption></figure>`).join("")}</div>` : ""}
		${wide.length ? `<div class="shots-wide">${wide.map((s) => `<figure class="window-frame"><div class="window-bar"><i></i><i></i><i></i></div><img src="${asset(p, s.src)}" alt="${escapeHtml(s.alt)}" loading="lazy"><figcaption>${escapeHtml(s.alt)}</figcaption></figure>`).join("")}</div>` : ""}
	</section>` : ""}
	<section class="container more-examples">
		<h2>More apps built with Giode</h2>
		<div class="mini-grid">${PROJECTS.filter((x) => x !== p).map((x) => `<a class="mini" href="#/examples/${x.slug}"><img src="${asset(x, "icon.png")}" alt="" width="40" height="40" loading="lazy"><span><b>${escapeHtml(x.name)}</b><small>${escapeHtml(x.category)}</small></span></a>`).join("")}</div>
	</section>`;
	fillDownloads(view.querySelector("#download"), p, repoURL);
}

async function fillDownloads(box, p, repoURL) {
	const { os, arm } = detectOS();
	const supported = p.platforms.includes(os);
	const releases = `${repoURL}/releases`;
	let rel;
	try {
		rel = await latestRelease(p.repo);
	} catch (e) {
		box.innerHTML = downloadFallback(p, os, releases, `Couldn't reach GitHub (${escapeHtml(e.message)}).`);
		return;
	}
	if (!rel || !rel.assets.length) {
		box.innerHTML = downloadFallback(p, os, releases, "There's no published release yet.");
		return;
	}
	const by = pick(rel.assets, arm);
	// Only the project's own platforms: a stray file in a release
	// shouldn't offer an app for a system it doesn't support.
	for (const pl of Object.keys(by)) {
		if (!p.platforms.includes(pl)) delete by[pl];
	}
	const best = supported && by[os] && by[os][0];
	let primary;
	if (best && os === "ios") {
		primary = `<div class="dl-primary"><div class="dl-note">${platformIcon("ios")} iOS apps install from the App Store or TestFlight. The release has the IPA for sideloading or your own signing.</div>
			<a class="btn btn-primary" href="${best.url}">Download ${escapeHtml(best.name)}</a></div>`;
	} else if (best) {
		primary = `<div class="dl-primary">
			<a class="btn btn-primary btn-lg" href="${best.url}">
				${platformIcon(os)}<span>Download for ${PLATFORM_NAMES[os]}<small>${escapeHtml(fileKind(best.name))} · ${escapeHtml(best.name)}${best.size ? " · " + formatSize(best.size) : ""}</small></span>
			</a>
			<span class="dl-version">${escapeHtml(rel.name)}${rel.prerelease ? ' <em>pre-release</em>' : ""}${rel.date ? " · " + new Date(rel.date).toLocaleDateString() : ""}</span>
		</div>`;
	} else {
		const why = !os ? "We couldn't tell which system you're on."
			: supported ? `This release has no ${PLATFORM_NAMES[os]} download yet.`
			: `${escapeHtml(p.name)} isn't available for ${PLATFORM_NAMES[os]}; it runs on ${p.platforms.map((x) => PLATFORM_NAMES[x]).join(", ")}.`;
		primary = `<div class="dl-primary"><div class="dl-note">${why}</div></div>`;
	}
	const others = Object.entries(by).flatMap(([pl, files]) => files.map((f) => ({ pl, f }))).filter(({ f }) => !best || f.url !== best.url);
	box.innerHTML = primary + (others.length ? `<details class="dl-others"${best ? "" : " open"}>
		<summary>Other downloads</summary>
		<ul>${others.map(({ pl, f }) => `<li><a href="${f.url}">${platformIcon(pl)}<span><b>${PLATFORM_NAMES[pl]}</b> · ${escapeHtml(fileKind(f.name))}</span><code>${escapeHtml(f.name)}</code><small>${formatSize(f.size)}</small></a></li>`).join("")}</ul>
		<a class="all-releases" href="${releases}" target="_blank" rel="noopener">All releases on GitHub →</a>
	</details>` : `<a class="all-releases" href="${releases}" target="_blank" rel="noopener">All releases on GitHub →</a>`);
}

function downloadFallback(p, os, releases, reason) {
	const sys = os ? PLATFORM_NAMES[os] : "your system";
	return `<div class="dl-primary">
		<a class="btn btn-primary btn-lg" href="${releases}" target="_blank" rel="noopener">${os ? platformIcon(os) : ""}<span>Get ${escapeHtml(p.name)} for ${sys}<small>From the releases on GitHub</small></span></a>
		<div class="dl-note"><span>${reason} Builds are on the project's releases page; or build it from source with <code>giode build</code>.</span></div>
	</div>`;
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
