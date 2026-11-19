// Small inline icons: platforms and the home page's features.

const svg = (body, size = 16) => `<svg class="ico" viewBox="0 0 24 24" width="${size}" height="${size}" aria-hidden="true">${body}</svg>`;
const stroke = (d) => `<path d="${d}" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>`;

const PLATFORMS = {
	windows: '<path fill="currentColor" d="M3 5.5 10.5 4.5v7H3zM11.5 4.4 21 3v8.5h-9.5zM3 12.5h7.5v7L3 18.5zM11.5 12.5H21V21l-9.5-1.4z"/>',
	macos: '<path fill="currentColor" d="M16.4 12.6c0-2.5 2-3.6 2.1-3.7-1.2-1.7-3-1.9-3.6-1.9-1.5-.2-3 .9-3.8.9-.8 0-2-.9-3.3-.9-1.7 0-3.3 1-4.1 2.5-1.8 3.1-.5 7.6 1.3 10.1.8 1.2 1.8 2.6 3.1 2.5 1.2 0 1.7-.8 3.2-.8s1.9.8 3.2.8c1.3 0 2.2-1.2 3-2.4.9-1.4 1.3-2.7 1.3-2.8 0 0-2.4-.9-2.4-3.8ZM14 5.4c.7-.8 1.1-1.9 1-3-1 0-2.1.7-2.8 1.5-.6.7-1.2 1.8-1 2.9 1.1.1 2.1-.6 2.8-1.4Z"/>',
	linux: stroke("M4 5h16v14H4zM8 10l3 2-3 2M13 15h3"),
	android: '<path fill="currentColor" d="M6 10a6 6 0 0 1 12 0zM9 7.2a.8.8 0 1 0 0-1.6.8.8 0 0 0 0 1.6Zm6 0a.8.8 0 1 0 0-1.6.8.8 0 0 0 0 1.6ZM6 11h12v6.5a1.5 1.5 0 0 1-1.5 1.5H15v2.2a1.2 1.2 0 0 1-2.4 0V19h-1.2v2.2a1.2 1.2 0 0 1-2.4 0V19H7.5A1.5 1.5 0 0 1 6 17.5zM3.8 11a1.2 1.2 0 0 1 1.2 1.2v4.6a1.2 1.2 0 0 1-2.4 0v-4.6A1.2 1.2 0 0 1 3.8 11Zm16.4 0a1.2 1.2 0 0 1 1.2 1.2v4.6a1.2 1.2 0 0 1-2.4 0v-4.6a1.2 1.2 0 0 1 1.2-1.2ZM7.3 2.6l1.6 2.4M16.7 2.6l-1.6 2.4" stroke="currentColor" stroke-width="1" stroke-linecap="round"/>',
	ios: stroke("M8 2.8h8a1.8 1.8 0 0 1 1.8 1.8v14.8a1.8 1.8 0 0 1-1.8 1.8H8a1.8 1.8 0 0 1-1.8-1.8V4.6A1.8 1.8 0 0 1 8 2.8ZM10.5 18h3"),
	web: stroke("M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18ZM3 12h18M12 3c2.5 2.6 3.7 5.6 3.7 9s-1.2 6.4-3.7 9c-2.5-2.6-3.7-5.6-3.7-9S9.5 5.6 12 3Z"),
};

export function platformIcon(name, size = 16) {
	return PLATFORMS[name] ? svg(PLATFORMS[name], size) : "";
}

const FEATURES = {
	style: stroke("M4 20l4-1 11-11a2.1 2.1 0 0 0-3-3L5 16zM14 6l3 3M4 4h5M4 8h3"),
	components: stroke("M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM16.5 13v7M13 16.5h7"),
	gpu: stroke("M13 2 4 14h7l-1 8 9-12h-7z"),
	platforms: stroke("M3 5h12v9H3zM6 18h6M9 14v4M17 8h4v12h-4zM18.5 17h1"),
	state: stroke("M5 12a7 7 0 0 1 12-5l2-2v6h-6l2.3-2.3A4.5 4.5 0 1 0 16.5 12M12 9v3l2 2"),
	services: stroke("M12 3a6 6 0 0 0-6 6c0 5-2 6-2 6h16s-2-1-2-6a6 6 0 0 0-6-6ZM10 19a2 2 0 0 0 4 0"),
	cli: stroke("M4 5h16v14H4zM7.5 9.5 10 12l-2.5 2.5M12.5 15H16"),
	ai: stroke("M12 3v2M12 19v2M3 12h2M19 12h2M7 7h10v10H7zM10 10h4v4h-4z"),
	raw: stroke("M8 6 3 12l5 6M16 6l5 6-5 6M14 4l-4 16"),
	arrow: stroke("M5 12h14M13 6l6 6-6 6"),
	globe: stroke("M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18ZM3 12h18M12 3c2.5 2.6 3.7 5.6 3.7 9s-1.2 6.4-3.7 9c-2.5-2.6-3.7-5.6-3.7-9S9.5 5.6 12 3Z"),
	external: stroke("M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5"),
	github: '<path fill="currentColor" d="M12 2a10 10 0 0 0-3.2 19.5c.5.1.7-.2.7-.5v-1.7c-2.8.6-3.4-1.3-3.4-1.3-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.3 1.1 2.9.8.1-.6.4-1.1.6-1.3-2.2-.3-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.6 9.6 0 0 1 5 0c1.9-1.3 2.8-1 2.8-1 .5 1.4.2 2.4.1 2.7.6.7 1 1.6 1 2.7 0 3.9-2.3 4.7-4.6 5 .4.3.7.9.7 1.9V21c0 .3.2.6.7.5A10 10 0 0 0 12 2Z"/>',
	copy: stroke("M9 9h11v11H9zM5 15V4h11"),
	check: stroke("M5 12.5l4.5 4.5L19 7"),
	spark: stroke("M12 3l2.2 5.8L20 11l-5.8 2.2L12 19l-2.2-5.8L4 11l5.8-2.2z"),
};

export function icon(name, size = 22) {
	return FEATURES[name] ? svg(FEATURES[name], size) : "";
}
