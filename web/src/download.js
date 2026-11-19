// The visitor's system, for picking the right install command.

// detectOS returns windows, macos, linux, android or ios, and whether the
// CPU looks like ARM.
export function detectOS() {
	const ua = navigator.userAgent || "";
	const platform = (navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || "";
	const arm = /arm|aarch64/i.test(ua) || /arm/i.test(platform);
	if (/android/i.test(ua)) return { os: "android", arm };
	// iPadOS reports itself as a Mac; touch gives it away.
	if (/iphone|ipad|ipod/i.test(ua) || (/mac/i.test(platform) && navigator.maxTouchPoints > 1)) return { os: "ios", arm: true };
	if (/win/i.test(platform) || /windows/i.test(ua)) return { os: "windows", arm };
	if (/mac/i.test(platform) || /mac os/i.test(ua)) return { os: "macos", arm };
	if (/linux|x11|cros/i.test(platform + ua)) return { os: "linux", arm };
	return { os: "", arm };
}
