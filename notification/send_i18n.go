package notification

import "github.com/xmhrnx/giode/i18n"

// The reply field's send button, in the languages of giode's other
// built-in strings.
func init() {
	for loc, t := range map[string]string{
		"en": "Send", "de": "Senden", "fr": "Envoyer", "es": "Enviar", "it": "Invia", "pt": "Enviar",
		"nl": "Verzenden", "sv": "Skicka", "da": "Send", "nb": "Send", "no": "Send", "fi": "Lähetä",
		"pl": "Wyślij", "cs": "Odeslat", "sk": "Odoslať", "hu": "Küldés", "ro": "Trimite", "ru": "Отправить",
		"uk": "Надіслати", "el": "Αποστολή", "tr": "Gönder", "fa": "ارسال", "ar": "إرسال", "he": "שליחה",
		"hi": "भेजें", "ja": "送信", "ko": "보내기", "zh": "发送", "zh-tw": "傳送", "zh-hk": "傳送",
		"id": "Kirim", "vi": "Gửi", "th": "ส่ง",
	} {
		i18n.AddBuiltin(loc, map[string]any{"giode.send": t})
	}
}

func sendLabel(a Action) string {
	if a.Send != "" {
		return a.Send
	}
	return i18n.T("giode.send")
}
