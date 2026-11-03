package multicast

import (
	"sync"

	"github.com/xmhrnx/giode/android"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_multicast/classes multicast_android.java
//go:generate jar cf multicast_android.jar -C $TEMP/giode_multicast/classes .

const class = "org/giode/multicast/multicast_android"

func acquire(tag string) (func(), error) {
	id, err := android.CallInt(class, "acquire", "(Landroid/content/Context;Ljava/lang/String;)I", android.Context(), tag)
	if err != nil {
		return func() {}, err
	}
	var once sync.Once
	return func() {
		once.Do(func() { android.CallVoid(class, "release", "(I)V", id) })
	}, nil
}
