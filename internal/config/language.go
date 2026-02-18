package config

// CJK language codes (first 3 chars of the code).
var cjkCodes = map[string]bool{
	"zho": true,
	"jpn": true,
	"kor": true,
	"chi": true,
	"zh":  true,
	"ja":  true,
	"ko":  true,
}

// IsCJK returns true if the language code represents Chinese, Japanese, or Korean.
func IsCJK(langCode string) bool {
	if len(langCode) > 3 {
		langCode = langCode[:3]
	}
	return cjkCodes[langCode]
}

// CPSForLang returns the default CPS limit for the given language.
func CPSForLang(langCode string) float64 {
	if IsCJK(langCode) {
		return 11
	}
	return 15
}

// CPLForLang returns the default characters-per-line limit for the given language.
func CPLForLang(langCode string) int {
	if IsCJK(langCode) {
		return 25
	}
	return 42
}
