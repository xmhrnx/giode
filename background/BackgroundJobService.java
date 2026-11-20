// SPDX-License-Identifier: Unlicense OR MIT

package org.giode.background;

import android.app.job.JobParameters;
import android.app.job.JobService;
import android.content.Context;

import java.util.HashMap;

// BackgroundJobService runs giode/background's tasks for JobScheduler,
// starting the app's Go side first when the process was started for
// the job.
public final class BackgroundJobService extends JobService {
	private static final HashMap<Integer, JobParameters> running = new HashMap<Integer, JobParameters>();
	private static BackgroundJobService instance;
	private static int next;

	static native void run(int rid, String name);
	static native void stop(int rid);

	@Override public boolean onStartJob(JobParameters params) {
		String name = params.getExtras().getString("name");
		if (name == null) {
			return false;
		}
		int rid;
		synchronized (running) {
			instance = this;
			rid = ++next;
			running.put(rid, params);
		}
		background_android.startGo(getApplicationContext());
		run(rid, name);
		return true;
	}

	@Override public boolean onStopJob(JobParameters params) {
		Integer found = null;
		synchronized (running) {
			for (java.util.Map.Entry<Integer, JobParameters> e : running.entrySet()) {
				if (e.getValue() == params) {
					found = e.getKey();
				}
			}
			if (found != null) {
				running.remove(found);
			}
		}
		if (found != null) {
			stop(found);
		}
		return true; // try again later
	}

	// done is called by Go when a task returns.
	static void done(int rid, boolean retry) {
		JobParameters params;
		BackgroundJobService svc;
		synchronized (running) {
			params = running.remove(rid);
			svc = instance;
		}
		if (params != null && svc != null) {
			svc.jobFinished(params, retry);
		}
	}
}
