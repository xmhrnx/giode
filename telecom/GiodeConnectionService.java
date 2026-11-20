package org.giode.telecom;

import android.os.Bundle;
import android.telecom.Connection;
import android.telecom.ConnectionRequest;
import android.telecom.ConnectionService;
import android.telecom.DisconnectCause;
import android.telecom.PhoneAccountHandle;

// GiodeConnectionService makes the connections for giode/telecom's
// calls. The package's android_manifest.xml declares it.
public final class GiodeConnectionService extends ConnectionService {
	private static telecom_android.CallConnection make(ConnectionRequest r, Bundle extras) {
		if (extras == null) {
			extras = new Bundle();
		}
		String id = extras.getString(telecom_android.EXTRA_ID, "");
		String handle = extras.getString(telecom_android.EXTRA_HANDLE, "");
		String name = extras.getString(telecom_android.EXTRA_NAME, handle);
		boolean video = extras.getBoolean(telecom_android.EXTRA_VIDEO, false);
		return new telecom_android.CallConnection(id, handle, name, video);
	}

	@Override
	public Connection onCreateIncomingConnection(PhoneAccountHandle account, ConnectionRequest r) {
		if (telecom_android.app == null) {
			telecom_android.app = getApplicationContext();
		}
		Bundle extras = r.getExtras();
		Bundle inner = extras != null ? extras.getBundle(android.telecom.TelecomManager.EXTRA_INCOMING_CALL_EXTRAS) : null;
		telecom_android.CallConnection c = make(r, inner != null ? inner : extras);
		c.setRinging();
		return c;
	}

	@Override
	public Connection onCreateOutgoingConnection(PhoneAccountHandle account, ConnectionRequest r) {
		if (telecom_android.app == null) {
			telecom_android.app = getApplicationContext();
		}
		telecom_android.CallConnection c = make(r, r.getExtras());
		c.setDialing();
		telecom_android.onEvent("start", c.id, r.getAddress() != null ? r.getAddress().getSchemeSpecificPart() : "");
		telecom_android.onEvent("audio_on", "", "");
		return c;
	}

	@Override
	public void onCreateIncomingConnectionFailed(PhoneAccountHandle account, ConnectionRequest r) {
		Bundle e = r.getExtras();
		telecom_android.onEvent("end", e != null ? e.getString(telecom_android.EXTRA_ID, "") : "", "");
	}

	@Override
	public void onCreateOutgoingConnectionFailed(PhoneAccountHandle account, ConnectionRequest r) {
		Bundle e = r.getExtras();
		telecom_android.onEvent("end", e != null ? e.getString(telecom_android.EXTRA_ID, "") : "", "");
	}
}
