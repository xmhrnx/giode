// SPDX-License-Identifier: Unlicense OR MIT

package org.giode.background;

import android.app.job.JobInfo;
import android.app.job.JobScheduler;
import android.content.ComponentName;
import android.content.Context;
import android.os.PersistableBundle;

// background_android schedules giode/background's tasks with
// JobScheduler.
public final class background_android {
	static void startGo(Context ctx) {
		try {
			Class.forName("org.gioui.Gio").getMethod("init", Context.class).invoke(null, ctx);
		} catch (Exception e) {
		}
	}

	static int jobID(String name) {
		return ("giode.background." + name).hashCode();
	}

	public static String schedule(Context ctx, String name, long intervalMs, long delayMs, boolean network, boolean charging) {
		try {
			JobInfo.Builder b = new JobInfo.Builder(jobID(name), new ComponentName(ctx, BackgroundJobService.class));
			PersistableBundle extras = new PersistableBundle();
			extras.putString("name", name);
			b.setExtras(extras);
			if (intervalMs > 0) {
				b.setPeriodic(Math.max(intervalMs, JobInfo.getMinPeriodMillis()));
			} else if (delayMs > 0) {
				b.setMinimumLatency(delayMs);
			}
			if (network) {
				b.setRequiredNetworkType(JobInfo.NETWORK_TYPE_ANY);
			}
			b.setRequiresCharging(charging);
			b.setPersisted(true);
			JobScheduler js = (JobScheduler) ctx.getSystemService(Context.JOB_SCHEDULER_SERVICE);
			if (js.schedule(b.build()) != JobScheduler.RESULT_SUCCESS) {
				return "JobScheduler refused the job";
			}
			return null;
		} catch (Exception e) {
			return e.toString();
		}
	}

	public static void cancel(Context ctx, String name) {
		JobScheduler js = (JobScheduler) ctx.getSystemService(Context.JOB_SCHEDULER_SERVICE);
		js.cancel(jobID(name));
	}
}
