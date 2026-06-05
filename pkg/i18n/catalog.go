package i18n

// catalog maps each supported language to its key→string table. The per-language
// tables live in en.go / fr.go / es.go. English (en) is the source of truth and
// must define every key; the others may omit keys and T falls back to English.
var catalog = map[Lang]map[string]string{
	EN: en,
	FR: fr,
	ES: es,
}
