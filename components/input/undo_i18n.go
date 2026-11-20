package input

import "github.com/xmhrnx/giode/i18n"

// The edit menu's Undo and Redo, in the languages of giode's other
// built-in strings.
func init() {
	for loc, t := range map[string][2]string{
		"en": {"Undo", "Redo"}, "de": {"Rückgängig", "Wiederholen"}, "fr": {"Annuler", "Rétablir"},
		"es": {"Deshacer", "Rehacer"}, "it": {"Annulla", "Ripeti"}, "pt": {"Desfazer", "Refazer"},
		"pt-pt": {"Anular", "Refazer"}, "nl": {"Ongedaan maken", "Opnieuw"}, "sv": {"Ångra", "Gör om"},
		"da": {"Fortryd", "Annuller fortryd"}, "nb": {"Angre", "Gjør om"}, "no": {"Angre", "Gjør om"},
		"fi": {"Kumoa", "Tee uudelleen"}, "pl": {"Cofnij", "Ponów"}, "cs": {"Zpět", "Znovu"},
		"sk": {"Späť", "Znova"}, "sl": {"Razveljavi", "Uveljavi"}, "hr": {"Poništi", "Ponovi"},
		"hu": {"Visszavonás", "Mégis"}, "ro": {"Anulează", "Refă"}, "bg": {"Отмяна", "Повторение"},
		"ru": {"Отменить", "Повторить"}, "uk": {"Скасувати", "Повторити"}, "el": {"Αναίρεση", "Ακύρωση αναίρεσης"},
		"tr": {"Geri al", "Yinele"}, "lv": {"Atsaukt", "Atcelt atsaukšanu"}, "lt": {"Anuliuoti", "Grąžinti"},
		"et": {"Võta tagasi", "Tee uuesti"}, "fa": {"واگرد", "انجام دوباره"}, "ar": {"تراجع", "إعادة"},
		"he": {"ביטול", "ביצוע מחדש"}, "hi": {"पूर्ववत करें", "फिर से करें"}, "ja": {"元に戻す", "やり直す"},
		"ko": {"실행 취소", "다시 실행"}, "zh": {"撤销", "重做"}, "zh-tw": {"復原", "重做"}, "zh-hk": {"復原", "重做"},
		"id": {"Urungkan", "Ulangi"}, "vi": {"Hoàn tác", "Làm lại"}, "th": {"เลิกทำ", "ทำซ้ำ"},
	} {
		i18n.AddBuiltin(loc, map[string]any{"giode.undo": t[0], "giode.redo": t[1]})
	}
}
