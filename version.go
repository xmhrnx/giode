package giode

import "encoding/base64"

// AppVersion and AppBuild are the app's version and build number from
// giode.toml (app.version, app.build): the giode CLI sets them when it
// builds or runs a desktop app (with -ldflags -X). They're empty in a
// plain go build.
var (
	AppVersion string
	AppBuild   string
)

// AppName is the app's name from giode.toml (app.name), which the giode
// CLI builds in (as buildAppName, base64 encoded since names have
// spaces). The start-at-login entry shows it; without it, the window
// title is used.
var AppName string

// buildAppName is app.name in URL-safe base64, set by the giode CLI.
var buildAppName string

func init() {
	if AppName == "" && buildAppName != "" {
		if b, err := base64.URLEncoding.DecodeString(buildAppName); err == nil {
			AppName = string(b)
		}
	}
}
