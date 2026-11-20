package spellcheck

import (
	"strconv"
	"strings"

	"github.com/xmhrnx/giode/android"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_spellcheck/classes spellcheck_android.java
//go:generate jar cf spellcheck_android.jar -C $TEMP/giode_spellcheck/classes .

const class = "org/giode/spellcheck/spellcheck_android"

func available(lang string) bool {
	ok, _ := android.CallBool(class, "available", "(Landroid/content/Context;Ljava/lang/String;)Z", android.Context(), lang)
	return ok
}

func check(text, lang string) []Misspelling {
	out, err := android.CallString(class, "check", "(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;", android.Context(), text, lang)
	if err != nil {
		return nil
	}
	x := newRuneIndex(text)
	var ms []Misspelling
	for _, p := range strings.Split(out, ";") {
		a, b, ok := strings.Cut(p, ",")
		if !ok {
			continue
		}
		start, _ := strconv.Atoi(a)
		n, _ := strconv.Atoi(b)
		ms = append(ms, fromUTF16(text, x, start, n))
	}
	return ms
}

func suggest(word, lang string) []string {
	out, err := android.CallString(class, "suggest", "(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;", android.Context(), word, lang)
	if err != nil {
		return nil
	}
	var s []string
	for _, l := range strings.Split(out, "\n") {
		if l != "" {
			s = append(s, l)
		}
	}
	return s
}

// learn: Android's user dictionary needs a permission apps don't get;
// the word is ignored for the session.
func learn(string) {}
