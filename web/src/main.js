import "./style.css";
import { renderHome } from "./pages/home.js";
import { renderDocs, hasDoc } from "./pages/docs.js";
import { renderExamples, openProject } from "./pages/examples.js";
import { codeFor } from "./markdown.js";
import { selectTab } from "./widgets.js";

// The site is one page routed by the URL hash, so it works from any
// static host (GitHub Pages) and every page has a shareable link:
//
//	#/                      home
//	#/docs/<page>[#section] a docs page (Markdown under ../docs)
//	#/examples              apps built with Giode
//	#/examples/<slug>       redirects to that app's own website

const view = document.getElementById("view");

// parseHash splits "#/docs/window#drawing" into the route and the
// section to scroll to.
function parseHash() {
	const raw = decodeURIComponent(location.hash.replace(/^#\/?/, ""));
	const [path, section] = raw.split("#");
	return { parts: path.replace(/^\/+|\/+$/g, "").split("/").filter(Boolean), section: section || "" };
}

let lastPath = null;

async function route() {
	const { parts, section } = parseHash();
	const path = parts.join("/");
	const samePage = path === lastPath;
	lastPath = path;
	setActiveNav(parts[0] === "docs" ? (parts[1] === "recipes" ? "recipes" : parts[1] === "cli" ? "cli" : "docs") : parts[0] || "");
	closeNav();

	if (samePage && section) {
		scrollToSection(section);
		return;
	}
	document.body.dataset.page = parts[0] || "home";
	if (parts.length === 0) {
		renderHome(view);
	} else if (parts[0] === "docs") {
		await renderDocs(view, parts.slice(1).join("/") || "index");
	} else if (parts[0] === "examples") {
		if (parts[1]) {
			openProject(view, parts[1]);
			return;
		}
		else renderExamples(view);
	} else if (await hasDoc(path)) {
		// Links from before the redesign: #/styles, #/components/menu.
		location.replace("#/docs/" + path + (section ? "#" + section : ""));
		return;
	} else {
		view.innerHTML = `<section class="notfound"><h1>404</h1><p>There's nothing at <code>${path.replace(/</g, "&lt;")}</code>.</p><a class="btn btn-primary" href="#/">Go home</a></section>`;
		document.title = "Not found - Giode";
	}
	if (section) {
		requestAnimationFrame(() => scrollToSection(section));
	} else {
		window.scrollTo(0, 0);
	}
	view.focus({ preventScroll: true });
}

function scrollToSection(id) {
	const el = document.getElementById(id);
	if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
}

function setActiveNav(name) {
	document.querySelectorAll(".topnav a").forEach((a) => {
		a.classList.toggle("active", a.dataset.nav === name);
	});
}

// --- Theme -------------------------------------------------------------------

const themeBtn = document.getElementById("theme-btn");
const SUN = '<svg viewBox="0 0 24 24" width="19" height="19" aria-hidden="true"><g fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="4.5"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></g></svg>';
const MOON = '<svg viewBox="0 0 24 24" width="19" height="19" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z"/></svg>';

function currentTheme() {
	return document.documentElement.dataset.theme ||
		(matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark");
}

function paintThemeButton() {
	themeBtn.innerHTML = currentTheme() === "dark" ? SUN : MOON;
}

themeBtn.addEventListener("click", () => {
	const next = currentTheme() === "dark" ? "light" : "dark";
	document.documentElement.dataset.theme = next;
	try {
		localStorage.setItem("theme", next);
	} catch (e) {
		// not remembered
	}
	paintThemeButton();
});
paintThemeButton();

// --- Mobile navigation ---------------------------------------------------------

const navBtn = document.getElementById("nav-btn");

function closeNav() {
	document.body.classList.remove("nav-open");
	navBtn.setAttribute("aria-expanded", "false");
}

navBtn.addEventListener("click", () => {
	const open = document.body.classList.toggle("nav-open");
	navBtn.setAttribute("aria-expanded", String(open));
});

// --- Shared widgets (event delegation) ---------------------------------------

document.addEventListener("click", (e) => {
	const copy = e.target.closest("[data-copy], .copy");
	if (copy) {
		const text = copy.dataset.copy !== undefined ? copy.dataset.copy : codeFor(Number(copy.dataset.code));
		navigator.clipboard.writeText(text).then(() => {
			copy.classList.add("copied");
			const label = copy.querySelector(".copy-label") || copy;
			const old = label.textContent;
			label.textContent = "Copied";
			setTimeout(() => {
				label.textContent = old;
				copy.classList.remove("copied");
			}, 1400);
		});
		return;
	}
	const tab = e.target.closest(".tab[data-group]");
	if (tab) {
		selectTab(tab.dataset.group, tab.dataset.tab, tab.closest(".tabs, .install"));
	}
});


window.addEventListener("hashchange", route);
route();
