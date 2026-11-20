package datetime

import (
	"time"

	"github.com/xmhrnx/giode/i18n"
)

// Relative describes t from now: "now", "5 minutes ago", "in 3 hours",
// "yesterday", "tomorrow", "in 2 days", "3 weeks ago", "in 4 months",
// "2 years ago". Days count calendar days, so 23:00 yesterday is
// "yesterday" at 01:00. Its texts are i18n keys "giode.rel.*", with
// English, German, French, Spanish, Russian and Persian built in.
func (l Locale) Relative(t, now time.Time) string {
	d := t.Sub(now)
	abs := d
	if abs < 0 {
		abs = -abs
	}
	past := d < 0
	unit := func(name string, n int) string {
		key := "giode.rel." + name
		if past {
			key += "_ago"
		} else {
			key += "_in"
		}
		return l.Digits(i18n.N(key, n))
	}
	switch {
	case abs < 45*time.Second:
		return i18n.T("giode.rel.now")
	case abs < 45*time.Minute:
		return unit("minutes", max(1, int((abs+30*time.Second)/time.Minute)))
	case abs < 22*time.Hour && SameDay(t.In(now.Location()), now):
		return unit("hours", max(1, int((abs+30*time.Minute)/time.Hour)))
	}
	days := DaysBetween(now, t)
	switch {
	case days == 0:
		return unit("hours", max(1, int((abs+30*time.Minute)/time.Hour)))
	case days == -1:
		return i18n.T("giode.rel.yesterday")
	case days == 1:
		return i18n.T("giode.rel.tomorrow")
	}
	n := days
	if n < 0 {
		n = -n
	}
	switch {
	case n < 7:
		return unit("days", n)
	case n < 28:
		return unit("weeks", (n+3)/7)
	case n < 335:
		return unit("months", max(1, (n+15)/30))
	}
	return unit("years", max(1, (n+182)/365))
}

// Relative describes t from now in the current locale.
func Relative(t, now time.Time) string { return Current().Relative(t, now) }

func forms(one, other string) map[string]string {
	return map[string]string{"one": one, "other": other}
}

func ruForms(one, few, many string) map[string]string {
	return map[string]string{"one": one, "few": few, "many": many, "other": many}
}

func init() {
	i18n.AddBuiltin("en", map[string]any{
		"giode.rel.now": "now", "giode.rel.yesterday": "yesterday", "giode.rel.tomorrow": "tomorrow",
		"giode.rel.minutes_ago": forms("{n} minute ago", "{n} minutes ago"),
		"giode.rel.minutes_in":  forms("in {n} minute", "in {n} minutes"),
		"giode.rel.hours_ago":   forms("{n} hour ago", "{n} hours ago"),
		"giode.rel.hours_in":    forms("in {n} hour", "in {n} hours"),
		"giode.rel.days_ago":    forms("{n} day ago", "{n} days ago"),
		"giode.rel.days_in":     forms("in {n} day", "in {n} days"),
		"giode.rel.weeks_ago":   forms("{n} week ago", "{n} weeks ago"),
		"giode.rel.weeks_in":    forms("in {n} week", "in {n} weeks"),
		"giode.rel.months_ago":  forms("{n} month ago", "{n} months ago"),
		"giode.rel.months_in":   forms("in {n} month", "in {n} months"),
		"giode.rel.years_ago":   forms("{n} year ago", "{n} years ago"),
		"giode.rel.years_in":    forms("in {n} year", "in {n} years"),
		"giode.date.today":      "Today",
	})
	i18n.AddBuiltin("de", map[string]any{
		"giode.rel.now": "jetzt", "giode.rel.yesterday": "gestern", "giode.rel.tomorrow": "morgen",
		"giode.rel.minutes_ago": forms("vor {n} Minute", "vor {n} Minuten"),
		"giode.rel.minutes_in":  forms("in {n} Minute", "in {n} Minuten"),
		"giode.rel.hours_ago":   forms("vor {n} Stunde", "vor {n} Stunden"),
		"giode.rel.hours_in":    forms("in {n} Stunde", "in {n} Stunden"),
		"giode.rel.days_ago":    forms("vor {n} Tag", "vor {n} Tagen"),
		"giode.rel.days_in":     forms("in {n} Tag", "in {n} Tagen"),
		"giode.rel.weeks_ago":   forms("vor {n} Woche", "vor {n} Wochen"),
		"giode.rel.weeks_in":    forms("in {n} Woche", "in {n} Wochen"),
		"giode.rel.months_ago":  forms("vor {n} Monat", "vor {n} Monaten"),
		"giode.rel.months_in":   forms("in {n} Monat", "in {n} Monaten"),
		"giode.rel.years_ago":   forms("vor {n} Jahr", "vor {n} Jahren"),
		"giode.rel.years_in":    forms("in {n} Jahr", "in {n} Jahren"),
		"giode.date.today":      "Heute",
	})
	i18n.AddBuiltin("fr", map[string]any{
		"giode.rel.now": "maintenant", "giode.rel.yesterday": "hier", "giode.rel.tomorrow": "demain",
		"giode.rel.minutes_ago": forms("il y a {n} minute", "il y a {n} minutes"),
		"giode.rel.minutes_in":  forms("dans {n} minute", "dans {n} minutes"),
		"giode.rel.hours_ago":   forms("il y a {n} heure", "il y a {n} heures"),
		"giode.rel.hours_in":    forms("dans {n} heure", "dans {n} heures"),
		"giode.rel.days_ago":    forms("il y a {n} jour", "il y a {n} jours"),
		"giode.rel.days_in":     forms("dans {n} jour", "dans {n} jours"),
		"giode.rel.weeks_ago":   forms("il y a {n} semaine", "il y a {n} semaines"),
		"giode.rel.weeks_in":    forms("dans {n} semaine", "dans {n} semaines"),
		"giode.rel.months_ago":  forms("il y a {n} mois", "il y a {n} mois"),
		"giode.rel.months_in":   forms("dans {n} mois", "dans {n} mois"),
		"giode.rel.years_ago":   forms("il y a {n} an", "il y a {n} ans"),
		"giode.rel.years_in":    forms("dans {n} an", "dans {n} ans"),
		"giode.date.today":      "Aujourd’hui",
	})
	i18n.AddBuiltin("es", map[string]any{
		"giode.rel.now": "ahora", "giode.rel.yesterday": "ayer", "giode.rel.tomorrow": "mañana",
		"giode.rel.minutes_ago": forms("hace {n} minuto", "hace {n} minutos"),
		"giode.rel.minutes_in":  forms("dentro de {n} minuto", "dentro de {n} minutos"),
		"giode.rel.hours_ago":   forms("hace {n} hora", "hace {n} horas"),
		"giode.rel.hours_in":    forms("dentro de {n} hora", "dentro de {n} horas"),
		"giode.rel.days_ago":    forms("hace {n} día", "hace {n} días"),
		"giode.rel.days_in":     forms("dentro de {n} día", "dentro de {n} días"),
		"giode.rel.weeks_ago":   forms("hace {n} semana", "hace {n} semanas"),
		"giode.rel.weeks_in":    forms("dentro de {n} semana", "dentro de {n} semanas"),
		"giode.rel.months_ago":  forms("hace {n} mes", "hace {n} meses"),
		"giode.rel.months_in":   forms("dentro de {n} mes", "dentro de {n} meses"),
		"giode.rel.years_ago":   forms("hace {n} año", "hace {n} años"),
		"giode.rel.years_in":    forms("dentro de {n} año", "dentro de {n} años"),
		"giode.date.today":      "Hoy",
	})
	i18n.AddBuiltin("ru", map[string]any{
		"giode.rel.now": "сейчас", "giode.rel.yesterday": "вчера", "giode.rel.tomorrow": "завтра",
		"giode.rel.minutes_ago": ruForms("{n} минуту назад", "{n} минуты назад", "{n} минут назад"),
		"giode.rel.minutes_in":  ruForms("через {n} минуту", "через {n} минуты", "через {n} минут"),
		"giode.rel.hours_ago":   ruForms("{n} час назад", "{n} часа назад", "{n} часов назад"),
		"giode.rel.hours_in":    ruForms("через {n} час", "через {n} часа", "через {n} часов"),
		"giode.rel.days_ago":    ruForms("{n} день назад", "{n} дня назад", "{n} дней назад"),
		"giode.rel.days_in":     ruForms("через {n} день", "через {n} дня", "через {n} дней"),
		"giode.rel.weeks_ago":   ruForms("{n} неделю назад", "{n} недели назад", "{n} недель назад"),
		"giode.rel.weeks_in":    ruForms("через {n} неделю", "через {n} недели", "через {n} недель"),
		"giode.rel.months_ago":  ruForms("{n} месяц назад", "{n} месяца назад", "{n} месяцев назад"),
		"giode.rel.months_in":   ruForms("через {n} месяц", "через {n} месяца", "через {n} месяцев"),
		"giode.rel.years_ago":   ruForms("{n} год назад", "{n} года назад", "{n} лет назад"),
		"giode.rel.years_in":    ruForms("через {n} год", "через {n} года", "через {n} лет"),
		"giode.date.today":      "Сегодня",
	})
	i18n.AddBuiltin("fa", map[string]any{
		"giode.rel.now": "اکنون", "giode.rel.yesterday": "دیروز", "giode.rel.tomorrow": "فردا",
		"giode.rel.minutes_ago": forms("{n} دقیقه پیش", "{n} دقیقه پیش"),
		"giode.rel.minutes_in":  forms("{n} دقیقه بعد", "{n} دقیقه بعد"),
		"giode.rel.hours_ago":   forms("{n} ساعت پیش", "{n} ساعت پیش"),
		"giode.rel.hours_in":    forms("{n} ساعت بعد", "{n} ساعت بعد"),
		"giode.rel.days_ago":    forms("{n} روز پیش", "{n} روز پیش"),
		"giode.rel.days_in":     forms("{n} روز بعد", "{n} روز بعد"),
		"giode.rel.weeks_ago":   forms("{n} هفته پیش", "{n} هفته پیش"),
		"giode.rel.weeks_in":    forms("{n} هفته بعد", "{n} هفته بعد"),
		"giode.rel.months_ago":  forms("{n} ماه پیش", "{n} ماه پیش"),
		"giode.rel.months_in":   forms("{n} ماه بعد", "{n} ماه بعد"),
		"giode.rel.years_ago":   forms("{n} سال پیش", "{n} سال پیش"),
		"giode.rel.years_in":    forms("{n} سال بعد", "{n} سال بعد"),
		"giode.date.today":      "امروز",
	})
}
