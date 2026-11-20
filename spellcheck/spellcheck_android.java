package org.giode.spellcheck;

import android.content.Context;
import android.os.Handler;
import android.os.HandlerThread;
import android.view.textservice.SentenceSuggestionsInfo;
import android.view.textservice.SpellCheckerSession;
import android.view.textservice.SuggestionsInfo;
import android.view.textservice.TextInfo;
import android.view.textservice.TextServicesManager;

import java.util.Locale;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

// spellcheck_android runs giode/spellcheck on the system's spell checker
// service. Sessions call back on the looper they're made on, so they
// live on their own thread; Go waits for each answer.
public final class spellcheck_android {
	private static HandlerThread thread;
	private static Handler handler;
	private static SpellCheckerSession session;
	private static String sessionLang;
	private static volatile SentenceSuggestionsInfo[] result;
	private static volatile CountDownLatch latch;

	private static final SpellCheckerSession.SpellCheckerSessionListener listener = new SpellCheckerSession.SpellCheckerSessionListener() {
		public void onGetSuggestions(SuggestionsInfo[] results) {}

		public void onGetSentenceSuggestions(SentenceSuggestionsInfo[] results) {
			result = results;
			CountDownLatch l = latch;
			if (l != null) {
				l.countDown();
			}
		}
	};

	private static Locale locale(String tag) {
		return tag == null || tag.isEmpty() ? Locale.getDefault() : Locale.forLanguageTag(tag);
	}

	// ask runs a sentence check of text and waits for the answer.
	@SuppressWarnings("deprecation")
	private static synchronized SentenceSuggestionsInfo[] ask(final Context ctx, final String text, final String lang, final int limit) {
		if (thread == null) {
			thread = new HandlerThread("giode-spellcheck");
			thread.start();
			handler = new Handler(thread.getLooper());
		}
		final CountDownLatch l = new CountDownLatch(1);
		latch = l;
		result = null;
		handler.post(new Runnable() {
			public void run() {
				try {
					if (session == null || !lang.equals(sessionLang)) {
						if (session != null) {
							session.close();
						}
						TextServicesManager tsm = (TextServicesManager) ctx.getSystemService(Context.TEXT_SERVICES_MANAGER_SERVICE);
						session = tsm == null ? null : tsm.newSpellCheckerSession(null, locale(lang), listener, lang.isEmpty());
						sessionLang = lang;
					}
					if (session == null) {
						l.countDown();
						return;
					}
					session.getSentenceSuggestions(new TextInfo[]{new TextInfo(text)}, limit);
				} catch (Exception e) {
					l.countDown();
				}
			}
		});
		try {
			l.await(3, TimeUnit.SECONDS);
		} catch (InterruptedException e) {
			return null;
		}
		return result;
	}

	public static boolean available(Context ctx, String lang) {
		TextServicesManager tsm = (TextServicesManager) ctx.getSystemService(Context.TEXT_SERVICES_MANAGER_SERVICE);
		return tsm != null && ask(ctx, "test", lang, 1) != null;
	}

	// check returns the typos as "start,length;" in UTF-16 offsets.
	public static String check(Context ctx, String text, String lang) {
		SentenceSuggestionsInfo[] res = ask(ctx, text, lang, 1);
		StringBuilder sb = new StringBuilder();
		if (res == null) {
			return "";
		}
		for (SentenceSuggestionsInfo s : res) {
			if (s == null) {
				continue;
			}
			for (int i = 0; i < s.getSuggestionsCount(); i++) {
				SuggestionsInfo si = s.getSuggestionsInfoAt(i);
				if ((si.getSuggestionsAttributes() & SuggestionsInfo.RESULT_ATTR_LOOKS_LIKE_TYPO) != 0) {
					sb.append(s.getOffsetAt(i)).append(',').append(s.getLengthAt(i)).append(';');
				}
			}
		}
		return sb.toString();
	}

	// suggest returns corrections for word, one per line.
	public static String suggest(Context ctx, String word, String lang) {
		SentenceSuggestionsInfo[] res = ask(ctx, word, lang, 6);
		StringBuilder sb = new StringBuilder();
		if (res == null) {
			return "";
		}
		for (SentenceSuggestionsInfo s : res) {
			if (s == null) {
				continue;
			}
			for (int i = 0; i < s.getSuggestionsCount(); i++) {
				SuggestionsInfo si = s.getSuggestionsInfoAt(i);
				for (int j = 0; j < si.getSuggestionsCount(); j++) {
					sb.append(si.getSuggestionAt(j)).append('\n');
				}
			}
		}
		return sb.toString();
	}
}
