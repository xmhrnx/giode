package mobile

import (
	"archive/zip"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestLifecycleHandlers(t *testing.T) {
	var bg, fg int
	SetLifecycleHandlers(func() { bg++ }, func() { fg++ })
	notify(&onBackground)
	notify(&onForeground)
	notify(&onForeground)
	if bg != 1 || fg != 2 {
		t.Errorf("bg=%d fg=%d, want 1 2", bg, fg)
	}
	SetLifecycleHandlers(nil, nil)
	notify(&onBackground)
	if bg != 1 {
		t.Error("cleared handler still ran")
	}
}

// The components android_manifest.xml declares are in the jar, or the
// app would crash at start.
func TestManifestClassesInJar(t *testing.T) {
	manifest, err := os.ReadFile("android_manifest.xml")
	if err != nil {
		t.Fatal(err)
	}
	jar, err := zip.OpenReader("mobile_android.jar")
	if err != nil {
		t.Fatal(err)
	}
	defer jar.Close()
	classes := map[string]bool{}
	for _, f := range jar.File {
		classes[f.Name] = true
	}
	names := regexp.MustCompile(`android:name="(org\.giode\.[^"]+)"`).FindAllSubmatch(manifest, -1)
	if len(names) == 0 {
		t.Fatal("no components")
	}
	for _, m := range names {
		c := strings.ReplaceAll(string(m[1]), ".", "/") + ".class"
		if !classes[c] {
			t.Errorf("%s isn't in mobile_android.jar", c)
		}
	}
}
