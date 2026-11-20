package datetime

import "strings"

// Month and weekday names of the languages Giode carries; others fall
// back to English, or to what an app gives (SetMonthNames, i18n keys).

func split(s string) []string { return strings.Split(s, "|") }

// numbered returns the months as "1月", for languages that number them.
func numbered(suffix string) []string {
	out := make([]string, 12)
	for i := range out {
		out[i] = itoa(i+1) + suffix
	}
	return out
}

var builtinMonths = map[string]*monthNames{
	"gregorian/en": {
		wide:  split("January|February|March|April|May|June|July|August|September|October|November|December"),
		short: split("Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec"),
	},
	"gregorian/de": {
		wide:  split("Januar|Februar|März|April|Mai|Juni|Juli|August|September|Oktober|November|Dezember"),
		short: split("Jan.|Feb.|März|Apr.|Mai|Juni|Juli|Aug.|Sept.|Okt.|Nov.|Dez."),
	},
	"gregorian/fr": {
		wide:  split("janvier|février|mars|avril|mai|juin|juillet|août|septembre|octobre|novembre|décembre"),
		short: split("janv.|févr.|mars|avr.|mai|juin|juil.|août|sept.|oct.|nov.|déc."),
	},
	"gregorian/es": {
		wide:  split("enero|febrero|marzo|abril|mayo|junio|julio|agosto|septiembre|octubre|noviembre|diciembre"),
		short: split("ene|feb|mar|abr|may|jun|jul|ago|sept|oct|nov|dic"),
	},
	"gregorian/it": {
		wide:  split("gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre"),
		short: split("gen|feb|mar|apr|mag|giu|lug|ago|set|ott|nov|dic"),
	},
	"gregorian/pt": {
		wide:  split("janeiro|fevereiro|março|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro"),
		short: split("jan.|fev.|mar.|abr.|mai.|jun.|jul.|ago.|set.|out.|nov.|dez."),
	},
	"gregorian/nl": {
		wide:  split("januari|februari|maart|april|mei|juni|juli|augustus|september|oktober|november|december"),
		short: split("jan|feb|mrt|apr|mei|jun|jul|aug|sep|okt|nov|dec"),
	},
	"gregorian/ru": {
		wide:     split("январь|февраль|март|апрель|май|июнь|июль|август|сентябрь|октябрь|ноябрь|декабрь"),
		short:    split("янв.|февр.|март|апр.|май|июнь|июль|авг.|сент.|окт.|нояб.|дек."),
		genitive: split("января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря"),
	},
	"gregorian/pl": {
		wide:     split("styczeń|luty|marzec|kwiecień|maj|czerwiec|lipiec|sierpień|wrzesień|październik|listopad|grudzień"),
		short:    split("sty|lut|mar|kwi|maj|cze|lip|sie|wrz|paź|lis|gru"),
		genitive: split("stycznia|lutego|marca|kwietnia|maja|czerwca|lipca|sierpnia|września|października|listopada|grudnia"),
	},
	"gregorian/tr": {
		wide:  split("Ocak|Şubat|Mart|Nisan|Mayıs|Haziran|Temmuz|Ağustos|Eylül|Ekim|Kasım|Aralık"),
		short: split("Oca|Şub|Mar|Nis|May|Haz|Tem|Ağu|Eyl|Eki|Kas|Ara"),
	},
	"gregorian/fa": {
		wide: split("ژانویه|فوریه|مارس|آوریل|مه|ژوئن|ژوئیه|اوت|سپتامبر|اکتبر|نوامبر|دسامبر"),
	},
	"gregorian/ar": {
		wide: split("يناير|فبراير|مارس|أبريل|مايو|يونيو|يوليو|أغسطس|سبتمبر|أكتوبر|نوفمبر|ديسمبر"),
	},
	"gregorian/ja": {wide: numbered("月")},
	"gregorian/zh": {wide: numbered("月")},
	"gregorian/ko": {wide: numbered("월")},
	"persian/fa": {
		wide: split("فروردین|اردیبهشت|خرداد|تیر|مرداد|شهریور|مهر|آبان|آذر|دی|بهمن|اسفند"),
	},
	"persian/en": {
		wide:  split("Farvardin|Ordibehesht|Khordad|Tir|Mordad|Shahrivar|Mehr|Aban|Azar|Dey|Bahman|Esfand"),
		short: split("Far|Ord|Kho|Tir|Mor|Sha|Meh|Aba|Aza|Dey|Bah|Esf"),
	},
}

// weekdayNames are wide, abbreviated and narrow names, Sunday first.
var weekdayNames = map[string][3][7]string{
	"en": {
		{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		{"S", "M", "T", "W", "T", "F", "S"},
	},
	"de": {
		{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
		{"So.", "Mo.", "Di.", "Mi.", "Do.", "Fr.", "Sa."},
		{"S", "M", "D", "M", "D", "F", "S"},
	},
	"fr": {
		{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"},
		{"dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."},
		{"D", "L", "M", "M", "J", "V", "S"},
	},
	"es": {
		{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"},
		{"dom", "lun", "mar", "mié", "jue", "vie", "sáb"},
		{"D", "L", "M", "X", "J", "V", "S"},
	},
	"it": {
		{"domenica", "lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato"},
		{"dom", "lun", "mar", "mer", "gio", "ven", "sab"},
		{"D", "L", "M", "M", "G", "V", "S"},
	},
	"pt": {
		{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"},
		{"dom.", "seg.", "ter.", "qua.", "qui.", "sex.", "sáb."},
		{"D", "S", "T", "Q", "Q", "S", "S"},
	},
	"nl": {
		{"zondag", "maandag", "dinsdag", "woensdag", "donderdag", "vrijdag", "zaterdag"},
		{"zo", "ma", "di", "wo", "do", "vr", "za"},
		{"Z", "M", "D", "W", "D", "V", "Z"},
	},
	"ru": {
		{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"},
		{"вс", "пн", "вт", "ср", "чт", "пт", "сб"},
		{"В", "П", "В", "С", "Ч", "П", "С"},
	},
	"pl": {
		{"niedziela", "poniedziałek", "wtorek", "środa", "czwartek", "piątek", "sobota"},
		{"niedz.", "pon.", "wt.", "śr.", "czw.", "pt.", "sob."},
		{"N", "P", "W", "Ś", "C", "P", "S"},
	},
	"tr": {
		{"Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi"},
		{"Paz", "Pzt", "Sal", "Çar", "Per", "Cum", "Cmt"},
		{"P", "P", "S", "Ç", "P", "C", "C"},
	},
	"fa": {
		{"یکشنبه", "دوشنبه", "سه‌شنبه", "چهارشنبه", "پنجشنبه", "جمعه", "شنبه"},
		{"یکشنبه", "دوشنبه", "سه‌شنبه", "چهارشنبه", "پنجشنبه", "جمعه", "شنبه"},
		{"ی", "د", "س", "چ", "پ", "ج", "ش"},
	},
	"ar": {
		{"الأحد", "الاثنين", "الثلاثاء", "الأربعاء", "الخميس", "الجمعة", "السبت"},
		{"الأحد", "الاثنين", "الثلاثاء", "الأربعاء", "الخميس", "الجمعة", "السبت"},
		{"ح", "ن", "ث", "ر", "خ", "ج", "س"},
	},
	"ja": {
		{"日曜日", "月曜日", "火曜日", "水曜日", "木曜日", "金曜日", "土曜日"},
		{"日", "月", "火", "水", "木", "金", "土"},
		{"日", "月", "火", "水", "木", "金", "土"},
	},
	"zh": {
		{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"},
		{"周日", "周一", "周二", "周三", "周四", "周五", "周六"},
		{"日", "一", "二", "三", "四", "五", "六"},
	},
	"ko": {
		{"일요일", "월요일", "화요일", "수요일", "목요일", "금요일", "토요일"},
		{"일", "월", "화", "수", "목", "금", "토"},
		{"일", "월", "화", "수", "목", "금", "토"},
	},
}

// dayPeriods are the 12-hour clock's AM and PM.
var dayPeriods = map[string][2]string{
	"en": {"AM", "PM"},
	"es": {"a. m.", "p. m."},
	"ar": {"ص", "م"},
	"fa": {"ق.ظ.", "ب.ظ."},
	"ja": {"午前", "午後"},
	"zh": {"上午", "下午"},
	"ko": {"오전", "오후"},
	"tr": {"ÖÖ", "ÖS"},
	"hi": {"am", "pm"},
}
