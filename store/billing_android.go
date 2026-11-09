package store

import (
	"git.wow.st/gmp/jni"

	"gioui.org/app"
)

// playInstallers are the package names Google Play installs apps as.
var playInstallers = map[string]bool{"com.android.vending": true, "com.google.android.feedback": true}

func billingRequired() bool {
	var installer string
	err := jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
		ctx := jni.Object(app.AppContext())
		ctxCls := jni.GetObjectClass(env, ctx)
		pm, err := jni.CallObjectMethod(env, ctx, jni.GetMethodID(env, ctxCls, "getPackageManager", "()Landroid/content/pm/PackageManager;"))
		if err != nil {
			return err
		}
		name, err := jni.CallObjectMethod(env, ctx, jni.GetMethodID(env, ctxCls, "getPackageName", "()Ljava/lang/String;"))
		if err != nil {
			return err
		}
		// Deprecated since Android 11 for getInstallSourceInfo, but still
		// answers, on every version.
		inst, err := jni.CallObjectMethod(env, pm, jni.GetMethodID(env, jni.GetObjectClass(env, pm),
			"getInstallerPackageName", "(Ljava/lang/String;)Ljava/lang/String;"), jni.Value(name))
		if err != nil {
			return err
		}
		installer = jni.GoString(env, jni.String(inst))
		return nil
	})
	return err == nil && playInstallers[installer]
}
