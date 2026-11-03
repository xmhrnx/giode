package org.giode.multicast;

import android.content.Context;
import android.net.wifi.WifiManager;

import java.util.HashMap;
import java.util.Map;

// multicast_android holds WifiManager multicast locks by id.
public final class multicast_android {
	private static final Map<Integer, WifiManager.MulticastLock> locks = new HashMap<>();
	private static int next;

	// acquire returns the lock's id, or 0 when the device has no Wi-Fi.
	public static synchronized int acquire(Context ctx, String tag) {
		WifiManager wm = (WifiManager) ctx.getApplicationContext().getSystemService(Context.WIFI_SERVICE);
		if (wm == null) {
			return 0;
		}
		WifiManager.MulticastLock l = wm.createMulticastLock(tag);
		l.setReferenceCounted(false);
		l.acquire();
		next++;
		locks.put(next, l);
		return next;
	}

	public static synchronized void release(int id) {
		WifiManager.MulticastLock l = locks.remove(id);
		if (l != null && l.isHeld()) {
			l.release();
		}
	}
}
