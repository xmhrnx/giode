// The Giode support relay: a Cloudflare Worker that turns the problem
// reports apps send with giode/support into GitHub issues, so users need
// no GitHub account and the GitHub token stays here, never in an app.
// See README.md to deploy it.
//
// Settings (wrangler.jsonc "vars", and secrets):
//   APPS          JSON: app name (support.Options.App) to "owner/repo".
//   GITHUB_TOKEN  secret: a fine-grained token with Issues read and write
//                 on those repositories.
//   ADMIN_TOKEN   secret: what reading a reporter's email takes.
//   KEY           optional secret: the key apps must send
//                 (support.Options.Key).
//   LABELS        optional, comma-separated: labels for every report;
//                 default "user report".
//   PER_HOUR, PER_DAY  optional: reports one address may send; 5 and 20.
// Binding: SUPPORT, a KV namespace for the rate limits and the emails.
//
// Every answer is JSON. A failure the reporter can do something about
// (too many reports, a summary too long) says so in "error"; any other
// failure answers with only a reason code in "code" (and "error"), which
// the app shows, and logs the details for Workers Logs.

const LIMITS = { title: 120, description: 20000, attachment: 30000, email: 254 };
const EMAIL_DAYS = 365;
const OS_NAMES = { darwin: "macOS", ios: "iOS", windows: "Windows", linux: "Linux", android: "Android", freebsd: "FreeBSD", openbsd: "OpenBSD" };

export default {
	async fetch(request, env) {
		const url = new URL(request.url);
		let resp;
		try {
			resp = await route(request, url, env);
		} catch (e) {
			resp = fail("orbit", 500, e && e.stack ? e.stack : String(e));
		}
		let error;
		if (resp.status >= 400) {
			try {
				error = (await resp.clone().json()).error;
			} catch {}
		}
		console.log(JSON.stringify({ method: request.method, path: url.pathname, status: resp.status, error }));
		return resp;
	},
};

async function route(request, url, env) {
	switch (url.pathname) {
		case "/report":
			if (request.method !== "POST") {
				return fail("pebble", 405, `${request.method} /report: reports are sent with POST`);
			}
			return await report(request, env);
		case "/contact":
			return await contact(request, url, env);
		case "/":
			return json(status(env));
	}
	return fail("meadow", 404, `nothing at ${url.pathname}: reports go to POST /report (support.Options.ReportURL must end in /report)`);
}

// status reports whether the relay has what sending reports takes
// (the APPS list, GITHUB_TOKEN and the KV namespace), and nothing more.
function status(env) {
	let apps = {};
	try {
		apps = parseApps(env.APPS);
	} catch {}
	return { ok: Object.keys(apps).length > 0 && Boolean(env.GITHUB_TOKEN) && Boolean(env.SUPPORT) };
}

// configError returns what's missing from the relay's settings for
// sending reports, as a failure's reason code and details, or null.
function configError(env) {
	if (!env.GITHUB_TOKEN) {
		return ["ember", "the relay has no GITHUB_TOKEN secret: run npx wrangler secret put GITHUB_TOKEN"];
	}
	if (!env.SUPPORT) {
		return ["willow", "the relay has no SUPPORT KV namespace: add it to kv_namespaces in wrangler.jsonc and deploy again"];
	}
	return null;
}

// fail answers with only a reason code, and logs the details.
function fail(code, status, detail) {
	console.log(JSON.stringify({ code, detail }));
	return json({ error: code, code }, status);
}

async function report(request, env) {
	if (env.KEY && request.headers.get("X-Support-Key") !== env.KEY) {
		return fail("thistle", 403, "the report's X-Support-Key isn't the relay's KEY");
	}
	const length = Number(request.headers.get("Content-Length") || 0);
	if (length > 200000) {
		return json({ error: "the report is too large" }, 413);
	}
	let r;
	try {
		r = await request.json();
	} catch {
		return fail("velvet", 400, "the report isn't valid JSON");
	}
	const missing = configError(env);
	if (missing) {
		return fail(missing[0], 500, missing[1]);
	}
	let apps;
	try {
		apps = parseApps(env.APPS);
	} catch (e) {
		return fail("cobalt", 500, e.message);
	}
	const repo = typeof r.app === "string" ? apps[r.app] : undefined;
	if (!repo) {
		const known = Object.keys(apps);
		return fail("saffron", 400,
			`unknown app ${JSON.stringify(r.app)}: the relay's APPS lists ${known.length ? known.join(", ") : "no apps"}`);
	}
	const title = clean(r.title).trim();
	if (!title || title.length > LIMITS.title) {
		return json({ error: "the summary must be 1 to 120 characters" }, 400);
	}
	if (clean(r.description).length > LIMITS.description) {
		return json({ error: "the description is too long" }, 400);
	}
	const email = clean(r.email).trim();
	if (email && (email.length > LIMITS.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email))) {
		return json({ error: "the email address isn't valid" }, 400);
	}

	const ip = request.headers.get("CF-Connecting-IP") || "unknown";
	const limited = await rateLimit(env, ip);
	if (limited) {
		return json({ error: limited }, 429);
	}

	const labels = (env.LABELS || "user report").split(",").map((l) => l.trim()).filter(Boolean);
	if (r.crash) {
		labels.push("crash");
	}
	const fields = { title: `[Report] ${title}`, body: issueBody(r, Boolean(email)) };
	let issue;
	try {
		try {
			issue = await github(env, `repos/${repo}/issues`, { ...fields, labels });
		} catch (e) {
			// Labels the token can't apply: send the report without them.
			if (e.status !== 422 && e.status !== 403) {
				throw e;
			}
			issue = await github(env, `repos/${repo}/issues`, fields);
		}
	} catch (e) {
		if (!e.status) {
			throw e;
		}
		return fail(githubCode(e.status), 502, githubError(e, repo));
	}
	if (email) {
		await env.SUPPORT.put(`email:${repo}#${issue.number}`, email, { expirationTtl: EMAIL_DAYS * 86400 });
	}
	return json({ url: issue.html_url, number: issue.number }, 201);
}

// contact returns the email a report's sender left:
// GET /contact?repo=owner/repo&issue=12 with Authorization: Bearer ADMIN_TOKEN.
async function contact(request, url, env) {
	if (request.method !== "GET") {
		return json({ error: "use GET for /contact" }, 405);
	}
	if (!env.ADMIN_TOKEN) {
		return json({ error: "the relay has no ADMIN_TOKEN secret: run npx wrangler secret put ADMIN_TOKEN" }, 500);
	}
	if (request.headers.get("Authorization") !== `Bearer ${env.ADMIN_TOKEN}`) {
		return json({ error: "unauthorized: send Authorization: Bearer <ADMIN_TOKEN>" }, 401);
	}
	if (!env.SUPPORT) {
		return json({ error: configError(env)[1] }, 500);
	}
	const repo = url.searchParams.get("repo");
	const issue = url.searchParams.get("issue");
	const email = repo && issue ? await env.SUPPORT.get(`email:${repo}#${issue}`) : null;
	return email ? json({ email }) : json({ error: "no email for that report" }, 404);
}

// rateLimit counts a report from ip and returns why it's turned away, or
// "" when it's within the limits.
async function rateLimit(env, ip) {
	const now = Date.now();
	const hour = `rate:h:${ip}:${Math.floor(now / 3600000)}`;
	const day = `rate:d:${ip}:${Math.floor(now / 86400000)}`;
	const [h, d] = await Promise.all([env.SUPPORT.get(hour), env.SUPPORT.get(day)]);
	if (Number(h || 0) >= Number(env.PER_HOUR || 5) || Number(d || 0) >= Number(env.PER_DAY || 20)) {
		return "too many reports from this network; try again later";
	}
	await Promise.all([
		env.SUPPORT.put(hour, String(Number(h || 0) + 1), { expirationTtl: 3700 }),
		env.SUPPORT.put(day, String(Number(d || 0) + 1), { expirationTtl: 90000 }),
	]);
	return "";
}

function issueBody(r, hasEmail) {
	const parts = [];
	const description = quiet(clean(r.description).trim());
	parts.push(description || "_No details given._");

	const rows = [
		["App version", [r.version, r.build && `(${r.build})`].filter(Boolean).join(" ")],
		["System", [OS_NAMES[r.os] || r.os, r.os_version].filter(Boolean).join(" ")],
		["CPU", r.arch],
		["Device", r.model],
		["Locale", r.locale],
	].filter(([, v]) => v);
	if (rows.length) {
		parts.push("| | |\n| --- | --- |\n" + rows.map(([k, v]) => `| ${k} | ${cell(v)} |`).join("\n"));
	}
	if (r.crash) {
		parts.push(details("Crash", clean(r.crash).slice(-LIMITS.attachment)));
	}
	if (r.logs) {
		parts.push(details("Logs", clean(r.logs).slice(-LIMITS.attachment)));
	}
	parts.push(
		"<sub>Sent from the app" +
			(hasEmail ? "; the reporter left an email for replies (see the relay's /contact)." : ".") +
			"</sub>",
	);
	return parts.join("\n\n");
}

// details folds long text into a collapsed code block, fenced with more
// backticks than the text has in a row.
function details(summary, text) {
	const runs = text.match(/`+/g) || [];
	const fence = "`".repeat(Math.max(3, ...runs.map((s) => s.length + 1)));
	return `<details><summary>${summary}</summary>\n\n${fence}\n${text}\n${fence}\n\n</details>`;
}

// quiet keeps text from mentioning people or linking issues: a report
// is written by anyone.
function quiet(s) {
	return s.replace(/@(?=[A-Za-z0-9-])/g, "@​").replace(/#(?=\d)/g, "#​");
}

function cell(v) {
	return quiet(clean(String(v))).replace(/[|\n\r]/g, " ").slice(0, 200);
}

function clean(v) {
	// No NULs or other control characters but tabs and newlines.
	return typeof v === "string" ? v.replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/g, "") : "";
}

// parseApps reads APPS, throwing when it isn't a JSON object.
function parseApps(s) {
	let apps;
	try {
		apps = typeof s === "object" && s !== null ? s : JSON.parse(s || "{}");
	} catch (e) {
		throw new Error(`the relay's APPS isn't valid JSON (${e.message}): it maps app names to "owner/repo"`);
	}
	if (!apps || typeof apps !== "object" || Array.isArray(apps)) {
		throw new Error(`the relay's APPS must be a JSON object mapping app names to "owner/repo"`);
	}
	return apps;
}

function githubCode(status) {
	return { 401: "lantern", 403: "juniper", 404: "harbor", 410: "quartz" }[status] || "falcon";
}

// githubError explains why GitHub refused to open the issue.
function githubError(e, repo) {
	const why = e.message ? ` GitHub said: ${e.message}` : "";
	switch (e.status) {
		case 401:
			return `GitHub rejected the relay's GITHUB_TOKEN (401): it's wrong, expired or revoked; set a new one with npx wrangler secret put GITHUB_TOKEN.${why}`;
		case 403:
			return `GitHub refused to open an issue in ${repo} (403): the token needs Issues: Read and write on that repository (or GitHub is rate-limiting it).${why}`;
		case 404:
			return `GitHub can't find ${repo} (404): check the name in APPS, and that the token's repository access includes it.${why}`;
		case 410:
			return `${repo} has issues turned off (410): turn them on in the repository's settings.${why}`;
		default:
			return `GitHub answered ${e.status} for ${repo}.${why}`;
	}
}

async function github(env, path, body) {
	const resp = await fetch(`https://api.github.com/${path}`, {
		method: "POST",
		headers: {
			Authorization: `Bearer ${env.GITHUB_TOKEN}`,
			Accept: "application/vnd.github+json",
			"X-GitHub-Api-Version": "2022-11-28",
			"User-Agent": "giode-support-relay",
			"Content-Type": "application/json",
		},
		body: JSON.stringify(body),
	});
	if (!resp.ok) {
		const text = await resp.text();
		let message = text.slice(0, 500);
		try {
			const b = JSON.parse(text);
			message = [b.message, ...(b.errors || []).map((x) => x.message || x.code)].filter(Boolean).join("; ") || message;
		} catch {}
		const e = new Error(message);
		e.status = resp.status;
		throw e;
	}
	return resp.json();
}

function json(obj, status = 200) {
	return new Response(JSON.stringify(obj), {
		status,
		headers: { "Content-Type": "application/json" },
	});
}

export const _test = { issueBody, quiet, details };
