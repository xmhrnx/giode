# giode.toml

The project file. Only `app.name` and `app.id` are required.

```toml
[app]
name = "Pocket Agent"         # display name
id = "com.example.agent"      # Android package name and Apple bundle ID
version = "1.4.0"
build = 12                    # Android versionCode, CFBundleVersion: raise per upload; giode.AppVersion and giode.AppBuild at run time
icon = "assets/icon.png"      # square PNG, 1024x1024 for the stores
package = "."                 # main package
publisher = "Example Ltd"     # shown by installers and Windows' app list
description = "An AI agent in your pocket."
website = "https://example.com"
schemes = ["agent"]           # URL schemes the app opens
links = ["https://example.com/app/*"]   # universal links / App Links; see giode links
tags = ""                     # Go build tags
renderer = "opengl"           # Gio's OpenGL renderer on macOS and iOS (Config.Renderer)
gio_patches = true            # build against Gio with giode's patches (default true)
ldflags = ""                  # extra linker flags; {version}, {build} and {id} are replaced
output = "dist"

[android]
min_sdk = 21
target_sdk = 35
arch = ["arm64", "amd64"]     # default: arm, arm64, 386, amd64
permissions = ["network", "camera", "microphone", "android.permission.VIBRATE"]
keep_alive = true             # declares giode's foreground service for App.KeepAlive
gio_patches = true            # build against Gio with giode's Android fixes (default true)
notifications = true          # POST_NOTIFICATIONS for App.Notify (default true)
queries = ["com.android.chrome"]
manifest = '<uses-feature android:name="android.hardware.camera" android:required="false"/>'
application = '<meta-data android:name="example" android:value="1"/>'
application_attributes = { requestLegacyExternalStorage = "true" }  # on <application>
java = ["android/java"]       # the app's own Java sources, compiled in (see giode/android)
icon_foreground = "assets/android-foreground.png"  # adaptive icon layers (Android 8+): 108dp
icon_background = "#0f172a"   # canvases, artwork in the middle 66dp; background a PNG or color
icon_monochrome = "assets/android-monochrome.png" # themed icons (Android 13+), optional
keystore = "release.keystore" # used by giode build -release
keystore_pass_env = "GIODE_KEYSTORE_PASS"
google_services = "google-services.json"   # from the Firebase console, for giode/push
dependencies = ["com.squareup.okhttp3:okhttp:4.12.0"]  # Maven libraries built in
libraries = ["libs/vendor.aar"]   # local .aar and .jar files built in
