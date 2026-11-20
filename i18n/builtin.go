package i18n

// giode's own strings - window buttons, the edit menu, the copy
// button - translated for common languages. An app's message files
// override any of them by key ("giode.close": "..."), and add
// languages missing here.

// builtinKeys are the keys, in builtinText's column order.
var builtinKeys = [...]string{
	"giode.minimize", "giode.maximize", "giode.restore", "giode.close",
	"giode.cut", "giode.copy", "giode.paste", "giode.select_all",
	"giode.copied", "giode.dismiss", "giode.license", "giode.no_license",
}

var builtinText = map[string][len(builtinKeys)]string{
	"en":    {"Minimize", "Maximize", "Restore", "Close", "Cut", "Copy", "Paste", "Select all", "Copied", "Dismiss", "License", "No license file was found in this module."},
	"de":    {"Minimieren", "Maximieren", "Wiederherstellen", "Schließen", "Ausschneiden", "Kopieren", "Einfügen", "Alles auswählen", "Kopiert", "Schließen", "Lizenz", "In diesem Modul wurde keine Lizenzdatei gefunden."},
	"fr":    {"Réduire", "Agrandir", "Restaurer", "Fermer", "Couper", "Copier", "Coller", "Tout sélectionner", "Copié", "Fermer", "Licence", "Aucun fichier de licence n’a été trouvé dans ce module."},
	"es":    {"Minimizar", "Maximizar", "Restaurar", "Cerrar", "Cortar", "Copiar", "Pegar", "Seleccionar todo", "Copiado", "Cerrar", "Licencia", "No se encontró ningún archivo de licencia en este módulo."},
	"it":    {"Riduci a icona", "Ingrandisci", "Ripristina", "Chiudi", "Taglia", "Copia", "Incolla", "Seleziona tutto", "Copiato", "Chiudi", "Licenza", "Nessun file di licenza trovato in questo modulo."},
	"pt":    {"Minimizar", "Maximizar", "Restaurar", "Fechar", "Recortar", "Copiar", "Colar", "Selecionar tudo", "Copiado", "Fechar", "Licença", "Nenhum arquivo de licença foi encontrado neste módulo."},
	"pt-pt": {"Minimizar", "Maximizar", "Restaurar", "Fechar", "Cortar", "Copiar", "Colar", "Selecionar tudo", "Copiado", "Fechar", "Licença", "Não foi encontrado nenhum ficheiro de licença neste módulo."},
	"nl":    {"Minimaliseren", "Maximaliseren", "Vorig formaat", "Sluiten", "Knippen", "Kopiëren", "Plakken", "Alles selecteren", "Gekopieerd", "Sluiten", "Licentie", "Er is geen licentiebestand gevonden in deze module."},
	"sv":    {"Minimera", "Maximera", "Återställ", "Stäng", "Klipp ut", "Kopiera", "Klistra in", "Markera allt", "Kopierat", "Stäng", "Licens", "Ingen licensfil hittades i den här modulen."},
	"da":    {"Minimer", "Maksimer", "Gendan", "Luk", "Klip", "Kopiér", "Sæt ind", "Markér alt", "Kopieret", "Luk", "Licens", "Der blev ikke fundet nogen licensfil i dette modul."},
	"nb":    {"Minimer", "Maksimer", "Gjenopprett", "Lukk", "Klipp ut", "Kopier", "Lim inn", "Merk alt", "Kopiert", "Lukk", "Lisens", "Fant ingen lisensfil i denne modulen."},
	"fi":    {"Pienennä", "Suurenna", "Palauta", "Sulje", "Leikkaa", "Kopioi", "Liitä", "Valitse kaikki", "Kopioitu", "Sulje", "Lisenssi", "Tästä moduulista ei löytynyt lisenssitiedostoa."},
	"pl":    {"Minimalizuj", "Maksymalizuj", "Przywróć", "Zamknij", "Wytnij", "Kopiuj", "Wklej", "Zaznacz wszystko", "Skopiowano", "Zamknij", "Licencja", "W tym module nie znaleziono pliku licencji."},
	"cs":    {"Minimalizovat", "Maximalizovat", "Obnovit", "Zavřít", "Vyjmout", "Kopírovat", "Vložit", "Vybrat vše", "Zkopírováno", "Zavřít", "Licence", "V tomto modulu nebyl nalezen žádný soubor s licencí."},
	"sk":    {"Minimalizovať", "Maximalizovať", "Obnoviť", "Zavrieť", "Vystrihnúť", "Kopírovať", "Prilepiť", "Vybrať všetko", "Skopírované", "Zavrieť", "Licencia", "V tomto module sa nenašiel žiadny súbor s licenciou."},
	"sl":    {"Minimiraj", "Maksimiraj", "Obnovi", "Zapri", "Izreži", "Kopiraj", "Prilepi", "Izberi vse", "Kopirano", "Zapri", "Licenca", "V tem modulu ni bilo najdene datoteke z licenco."},
	"hr":    {"Minimiziraj", "Maksimiziraj", "Vrati", "Zatvori", "Izreži", "Kopiraj", "Zalijepi", "Odaberi sve", "Kopirano", "Zatvori", "Licenca", "U ovom modulu nije pronađena datoteka licence."},
	"hu":    {"Kis méret", "Teljes méret", "Előző méret", "Bezárás", "Kivágás", "Másolás", "Beillesztés", "Az összes kijelölése", "Másolva", "Bezárás", "Licenc", "Ebben a modulban nem található licencfájl."},
	"ro":    {"Minimizare", "Maximizare", "Restabilire", "Închidere", "Decupare", "Copiere", "Lipire", "Selectează tot", "Copiat", "Închidere", "Licență", "Nu s-a găsit niciun fișier de licență în acest modul."},
	"bg":    {"Минимизиране", "Максимизиране", "Възстановяване", "Затваряне", "Изрязване", "Копиране", "Поставяне", "Избиране на всичко", "Копирано", "Затваряне", "Лиценз", "В този модул не е намерен файл с лиценз."},
	"ru":    {"Свернуть", "Развернуть", "Восстановить", "Закрыть", "Вырезать", "Копировать", "Вставить", "Выделить всё", "Скопировано", "Закрыть", "Лицензия", "В этом модуле не найден файл лицензии."},
	"uk":    {"Згорнути", "Розгорнути", "Відновити", "Закрити", "Вирізати", "Копіювати", "Вставити", "Вибрати все", "Скопійовано", "Закрити", "Ліцензія", "У цьому модулі не знайдено файлу ліцензії."},
	"el":    {"Ελαχιστοποίηση", "Μεγιστοποίηση", "Επαναφορά", "Κλείσιμο", "Αποκοπή", "Αντιγραφή", "Επικόλληση", "Επιλογή όλων", "Αντιγράφηκε", "Κλείσιμο", "Άδεια χρήσης", "Δεν βρέθηκε αρχείο άδειας χρήσης σε αυτή την ενότητα."},
	"tr":    {"Simge durumuna küçült", "Ekranı kapla", "Önceki boyut", "Kapat", "Kes", "Kopyala", "Yapıştır", "Tümünü seç", "Kopyalandı", "Kapat", "Lisans", "Bu modülde lisans dosyası bulunamadı."},
	"lv":    {"Minimizēt", "Maksimizēt", "Atjaunot", "Aizvērt", "Izgriezt", "Kopēt", "Ielīmēt", "Atlasīt visu", "Nokopēts", "Aizvērt", "Licence", "Šajā modulī netika atrasts licences fails."},
	"lt":    {"Sumažinti", "Padidinti", "Atkurti", "Uždaryti", "Iškirpti", "Kopijuoti", "Įklijuoti", "Pažymėti viską", "Nukopijuota", "Uždaryti", "Licencija", "Šiame modulyje licencijos failas nerastas."},
	"et":    {"Minimeeri", "Maksimeeri", "Taasta", "Sule", "Lõika", "Kopeeri", "Kleebi", "Vali kõik", "Kopeeritud", "Sule", "Litsents", "Selles moodulis ei leitud litsentsifaili."},
	"fa":    {"کوچک کردن", "بزرگ کردن", "بازگردانی", "بستن", "برش", "کپی", "چسباندن", "انتخاب همه", "کپی شد", "بستن", "مجوز", "هیچ فایل مجوزی در این ماژول پیدا نشد."},
	"ar":    {"تصغير", "تكبير", "استعادة", "إغلاق", "قص", "نسخ", "لصق", "تحديد الكل", "تم النسخ", "إغلاق", "الترخيص", "لم يُعثر على ملف ترخيص في هذه الوحدة."},
	"he":    {"מזער", "הגדל", "שחזר", "סגור", "גזור", "העתק", "הדבק", "בחר הכול", "הועתק", "סגור", "רישיון", "לא נמצא קובץ רישיון במודול זה."},
	"hi":    {"छोटा करें", "बड़ा करें", "पुनर्स्थापित करें", "बंद करें", "काटें", "कॉपी करें", "चिपकाएँ", "सभी चुनें", "कॉपी किया गया", "बंद करें", "लाइसेंस", "इस मॉड्यूल में कोई लाइसेंस फ़ाइल नहीं मिली।"},
	"ja":    {"最小化", "最大化", "元のサイズに戻す", "閉じる", "切り取り", "コピー", "貼り付け", "すべて選択", "コピーしました", "閉じる", "ライセンス", "このモジュールにライセンス ファイルが見つかりませんでした。"},
	"ko":    {"최소화", "최대화", "이전 크기로", "닫기", "잘라내기", "복사", "붙여넣기", "모두 선택", "복사됨", "닫기", "라이선스", "이 모듈에서 라이선스 파일을 찾을 수 없습니다."},
	"zh":    {"最小化", "最大化", "还原", "关闭", "剪切", "复制", "粘贴", "全选", "已复制", "关闭", "许可证", "此模块中未找到许可证文件。"},
	"zh-tw": {"最小化", "最大化", "還原", "關閉", "剪下", "複製", "貼上", "全選", "已複製", "關閉", "授權", "此模組中找不到授權檔案。"},
	"zh-hk": {"最小化", "最大化", "還原", "關閉", "剪下", "複製", "貼上", "全選", "已複製", "關閉", "授權", "此模組中找不到授權檔案。"},
	"id":    {"Minimalkan", "Maksimalkan", "Pulihkan", "Tutup", "Potong", "Salin", "Tempel", "Pilih semua", "Disalin", "Tutup", "Lisensi", "Tidak ada berkas lisensi di modul ini."},
	"vi":    {"Thu nhỏ", "Phóng to", "Khôi phục", "Đóng", "Cắt", "Sao chép", "Dán", "Chọn tất cả", "Đã sao chép", "Đóng", "Giấy phép", "Không tìm thấy tệp giấy phép trong mô-đun này."},
	"th":    {"ย่อเล็กสุด", "ขยายใหญ่สุด", "คืนค่า", "ปิด", "ตัด", "คัดลอก", "วาง", "เลือกทั้งหมด", "คัดลอกแล้ว", "ปิด", "สัญญาอนุญาต", "ไม่พบไฟล์สัญญาอนุญาตในโมดูลนี้"},
}

// builtin holds builtinText as catalogs.
var builtin = func() map[string]map[string]message {
	out := make(map[string]map[string]message, len(builtinText))
	for loc, texts := range builtinText {
		c := make(map[string]message, len(texts))
		for i, t := range texts {
			c[builtinKeys[i]] = message{text: t}
		}
		out[loc] = c
	}
	// Norwegian: Bokmål for the bare language and Nynorsk's neighbor.
	out["no"] = out["nb"]
	return out
}()

// AddBuiltin adds messages to giode's own strings for a locale, like
// Add, but below the app's messages and without making the locale one
// of the app's (Locales). Giode's packages use it for their strings
// (dates, relative times); apps use Add or Load.
func AddBuiltin(loc string, messages map[string]any) {
	loc = normalize(loc)
	mu.Lock()
	c := builtin[loc]
	if c == nil {
		c = map[string]message{}
		builtin[loc] = c
	}
	for k, v := range messages {
		switch v := v.(type) {
		case string:
			c[k] = message{text: v}
		case map[string]string:
			c[k] = message{forms: v}
		}
	}
	mu.Unlock()
	changed()
}
