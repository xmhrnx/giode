# CLI commands

| Command | What it does |
| --- | --- |
| `giode new [dir]` | Creates a project: a counter app in `main.go`, `go.mod` with the latest giode, `giode.toml` set up for the chosen platforms, placeholder assets (`assets/icon.png`, `assets/README.md`), `.gitignore` and a README. In a terminal it asks first: a checklist of every platform (Windows, macOS, Linux, Android, iOS, Web - whether or not this computer builds them; arrow keys move, Space checks, Enter goes on), then the name, app ID, version, module path, publisher, description and website, each with a suggestion Enter takes. `-y` skips the questions; `-platforms android,ios`, `-id`, `-name` and `-version` answer them. |
| `giode build` (no target) | Builds for this computer's platform: `macos`, `windows` or `linux`, the same as `giode build desktop`. |
| `giode init` | Adds `giode.toml` to an existing Go module. |
| `giode build <target>...` | Builds into `dist/<platform>`. Targets: `android` (APK), `aab` (Play bundle), `ios` (IPA), `ios-sim`, `macos` (universal), `dmg`, `windows`, `windows-installer` (setup program), `msix`, `winget`, `linux`, `deb`, `rpm`, `appimage`, `flatpak`, `web`, `desktop` (this computer) and `all` (everything this computer can build). `-release` signs with your certificates. |
| `giode run [target]` | Builds for a target and runs it: `desktop` (default: `go build`, then the program), `linux`, `windows` (Wine off Windows), `windows-installer`, `macos`, `android` (builds the device's architecture, installs, launches, follows the log), `ios-sim`, `web` (serves `dist/web`). `-mcp` starts giode's MCP server so an AI agent can drive the app. |
| `giode devices` | Lists Android devices and booted iOS simulators. |
| `giode doctor` | Checks Go, the Android SDK, build tools, platform, NDK and JDK, Xcode, and Gio's Linux libraries, with install commands for what's missing. |
| `giode keystore` | Creates an Android release keystore and points `giode.toml` at it. |
| `giode cert` | Creates a self-signed Windows code signing certificate for testing and points `giode.toml` at it. |
| `giode sign <file>...` | Signs Windows executables and installers with the configured certificate. |
| `giode links` | Writes `.well-known/assetlinks.json` and `apple-app-site-association` for each host in `app.links`, to upload to your site (fingerprints from `android.keystore` and `android.fingerprints`, team from `ios.team_id`). |
| `giode update-key` | Creates the Ed25519 key pair signing update feeds: the private key in `update.key` (added to `.gitignore`), the public key in `[update]`. |
| `giode release [targets]` | Builds desktop targets and writes `dist/update/`: the downloads, `feed.json` and its signature, for apps using the giode `update` package. |
| `giode bump [major\|minor\|patch\|build]` | Raises `app.version` and always `app.build`. In CI, `$GIODE_VERSION` (a tag like `v1.4.2`) and `$GIODE_BUILD` override both. |
| `giode ci` | Writes `.github/workflows/release.yml`: on a version tag (or a published release, with `ci.trigger = "release"`), runs the tests, builds packages for the platforms in `app.platforms` (iOS included), signed with the repository's secrets when present, and attaches them to the GitHub release. |
| `giode licenses` | Writes the licenses of the app's Go modules (on any platform) and Go's to `assets/licenses.json` (or `app.licenses`, refreshed on every build) for `giode.LicensesView`. |
| `giode upgrade [version]` | Updates the project's giode dependency and tidies `go.mod`. |
| `giode manifest [android\|ios\|macos]` | Prints what `giode.toml` adds to the manifest and Info.plist. |
| `giode clean` | Removes `dist`. |
| `giode self-update` | Updates the CLI to the latest release. |

Every command takes `-C dir` to run on a project elsewhere, and build commands take `-x`
to print what they run. Targets have short names too: `apk`, `bundle`, `win`, `mac`,
`sim`, `installer`, `browser`.

`build`, `release`, `upgrade`, `self-update`, `licenses` and `clean` print how long they
took when they finish (`Done in 21.3s.`), and `build` and `release` each target's time next
to its output. `run` prints how long the build (and on a device, the install) took before
the app starts.
