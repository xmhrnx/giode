package org.giode.biometric;

import android.app.Activity;
import android.content.Context;
import android.content.DialogInterface;
import android.hardware.biometrics.BiometricManager;
import android.hardware.biometrics.BiometricPrompt;
import android.hardware.fingerprint.FingerprintManager;
import android.os.Build;
import android.os.CancellationSignal;
import android.view.View;

import java.util.concurrent.Executor;

// biometric_android shows BiometricPrompt and reports the outcome to
// onResult with the codes in biometric.go.
public final class biometric_android {
	static native void onResult(int id, int code, String msg);

	static final int OK = 0, CANCELED = 1, NOT_ENROLLED = 2, LOCKOUT = 3, FAILED = 4, UNAVAILABLE = 5;

	// available returns 0 for none, 4 (biometric) for a usable one.
	@SuppressWarnings("deprecation")
	public static int available(Context ctx) {
		if (Build.VERSION.SDK_INT < 28) {
			return 0;
		}
		if (Build.VERSION.SDK_INT >= 30) {
			BiometricManager bm = ctx.getSystemService(BiometricManager.class);
			return bm != null && bm.canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG | BiometricManager.Authenticators.BIOMETRIC_WEAK) == BiometricManager.BIOMETRIC_SUCCESS ? 4 : 0;
		}
		if (Build.VERSION.SDK_INT >= 29) {
			BiometricManager bm = ctx.getSystemService(BiometricManager.class);
			return bm != null && bm.canAuthenticate() == BiometricManager.BIOMETRIC_SUCCESS ? 4 : 0;
		}
		FingerprintManager fm = ctx.getSystemService(FingerprintManager.class);
		return fm != null && fm.isHardwareDetected() && fm.hasEnrolledFingerprints() ? 4 : 0;
	}

	@SuppressWarnings("deprecation")
	public static void authenticate(final View view, final int id, final String title, final String reason, final String cancel, final boolean passcode) {
		view.post(new Runnable() {
			public void run() {
				if (Build.VERSION.SDK_INT < 28) {
					onResult(id, UNAVAILABLE, "needs Android 9 or later");
					return;
				}
				Activity activity = (Activity) view.getContext();
				final Executor main = activity.getMainExecutor();
				BiometricPrompt.Builder b = new BiometricPrompt.Builder(activity).setTitle(title);
				if (reason != null && !reason.isEmpty()) {
					b.setSubtitle(reason);
				}
				if (passcode && Build.VERSION.SDK_INT >= 30) {
					b.setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG | BiometricManager.Authenticators.BIOMETRIC_WEAK | BiometricManager.Authenticators.DEVICE_CREDENTIAL);
				} else if (passcode && Build.VERSION.SDK_INT >= 29) {
					b.setDeviceCredentialAllowed(true);
				} else {
					b.setNegativeButton(cancel, main, new DialogInterface.OnClickListener() {
						public void onClick(DialogInterface d, int which) {
							onResult(id, CANCELED, "");
						}
					});
				}
				BiometricPrompt prompt;
				try {
					prompt = b.build();
				} catch (Throwable e) {
					onResult(id, UNAVAILABLE, e.toString());
					return;
				}
				prompt.authenticate(new CancellationSignal(), main, new BiometricPrompt.AuthenticationCallback() {
					@Override
					public void onAuthenticationSucceeded(BiometricPrompt.AuthenticationResult r) {
						onResult(id, OK, "");
					}

					@Override
					public void onAuthenticationError(int code, CharSequence msg) {
						String m = msg == null ? "" : msg.toString();
						switch (code) {
						case BiometricPrompt.BIOMETRIC_ERROR_CANCELED:
						case BiometricPrompt.BIOMETRIC_ERROR_USER_CANCELED:
							onResult(id, CANCELED, m);
							break;
						case BiometricPrompt.BIOMETRIC_ERROR_NO_BIOMETRICS:
							onResult(id, NOT_ENROLLED, m);
							break;
						case BiometricPrompt.BIOMETRIC_ERROR_LOCKOUT:
						case BiometricPrompt.BIOMETRIC_ERROR_LOCKOUT_PERMANENT:
							onResult(id, LOCKOUT, m);
							break;
						case BiometricPrompt.BIOMETRIC_ERROR_HW_NOT_PRESENT:
						case BiometricPrompt.BIOMETRIC_ERROR_HW_UNAVAILABLE:
							onResult(id, UNAVAILABLE, m);
							break;
						default:
							onResult(id, FAILED, m);
						}
					}
					// onAuthenticationFailed is one attempt that didn't
					// match; the prompt stays up for another.
				});
			}
		});
	}
}
