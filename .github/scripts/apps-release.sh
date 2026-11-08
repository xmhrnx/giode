#!/usr/bin/env bash
# Collects the latest release files of the apps built with Giode (the
# examples pages' list, web/src/projects.js) into this repository's
# "apps" release, so visitors can try them without a toolchain. Run by
# .github/workflows/apps-release.yml; needs gh, jq and node, and
# GH_TOKEN with write access to this repository's releases. APPS_TOKEN,
# when set, reads the apps' repositories instead (for private ones).
set -uo pipefail

# read calls gh with the token that reads the apps' repositories.
read_app() {
	GH_TOKEN="${APPS_TOKEN:-$GH_TOKEN}" gh "$@"
}

repo="${GITHUB_REPOSITORY:?}"
tag=apps
work="$(mktemp -d)"
notes="$work/notes.md"
uploaded="$work/uploaded"
: >"$uploaded"

# Files that are never the app itself: checksums, signatures, debug
# symbols, store bundles (as on the website's download buttons).
skip='(\.sha256|\.sha512|\.sig|\.asc|checksums?|sha256sums|\.txt|\.json|\.blockmap|symbols|\.aab)$|native-debug-symbols|anyprobe'

# The release, made once; it's never marked latest, so Giode's own
# versions stay the latest release.
if ! gh release view "$tag" -R "$repo" >/dev/null 2>&1; then
	gh release create "$tag" -R "$repo" --target main --latest=false \
		--title "Apps built with Giode" --notes "Collecting the apps…" || exit 1
fi

{
	echo "Installers and packages of the apps built with [Giode](https://github.com/$repo), taken from each app's latest release. Refreshed weekly: see each app's repository for its full history and release notes."
	echo
	echo "| App | Version | Platforms |"
	echo "| --- | --- | --- |"
} >"$notes"

failed=()
while IFS=$'\t' read -r slug name app platforms; do
	if ! release="$(read_app api "repos/$app/releases/latest" 2>"$work/err")"; then
		echo "::warning::$name ($app): no release found: $(cat "$work/err")"
		failed+=("$name")
		continue
	fi
	version="$(jq -r .tag_name <<<"$release")"
	url="$(jq -r .html_url <<<"$release")"
	count=0
	while IFS=$'\t' read -r id file; do
		if grep -Eiq "$skip" <<<"$file"; then
			continue
		fi
		# Named for the app, so files from different apps can't clash.
		out="$file"
		if ! grep -Fiq "${slug//-/}" <<<"${file//[-_ .]/}"; then
			out="$slug-$file"
		fi
		if ! read_app api -H "Accept: application/octet-stream" "repos/$app/releases/assets/$id" >"$work/$out"; then
			echo "::warning::$name: couldn't download $file"
			rm -f "$work/$out"
			continue
		fi
		if gh release upload "$tag" "$work/$out" -R "$repo" --clobber; then
			echo "$out" >>"$uploaded"
			count=$((count + 1))
		else
			echo "::warning::$name: couldn't upload $out"
		fi
		rm -f "$work/$out"
	done < <(jq -r '.assets[] | [.id, .name] | @tsv' <<<"$release")
	if [ "$count" -eq 0 ]; then
		failed+=("$name")
	fi
	echo "| [$name](https://github.com/$app) | [$version]($url) | $platforms |" >>"$notes"
	echo "$name $version: $count file(s)"
done < <(node -e '
	const names = {windows: "Windows", macos: "macOS", linux: "Linux", android: "Android", ios: "iOS"};
	import(process.cwd() + "/web/src/projects.js").then(({PROJECTS}) => {
		for (const p of PROJECTS) {
			const platforms = Object.keys(names).filter((k) => p.platforms.includes(k)).map((k) => names[k]).join(", ");
			console.log([p.slug, p.name, p.repo, platforms].join("\t"));
		}
	});
')

# Files a previous run uploaded that no app has any more.
gh api "repos/$repo/releases/tags/$tag" --jq '.assets[] | [.id, .name] | @tsv' |
	while IFS=$'\t' read -r id file; do
		if ! grep -Fxq "$file" "$uploaded"; then
			echo "Removing $file"
			gh api -X DELETE "repos/$repo/releases/assets/$id"
		fi
	done

if [ "${#failed[@]}" -gt 0 ]; then
	printf '\nNot included this time: %s.\n' "$(IFS=,; echo "${failed[*]}" | sed 's/,/, /g')" >>"$notes"
fi
gh release edit "$tag" -R "$repo" --notes-file "$notes" --latest=false
cat "$notes" >>"${GITHUB_STEP_SUMMARY:-/dev/null}"
