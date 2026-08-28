package ingestion

var countryPrimaryLang = map[string][2]byte{
	"AU": {'e', 'n'},
	"BR": {'p', 't'},
	"CA": {'e', 'n'},
	"DE": {'d', 'e'},
	"ES": {'e', 's'},
	"FR": {'f', 'r'},
	"GB": {'e', 'n'},
	"IN": {'h', 'i'},
	"IT": {'i', 't'},
	"JP": {'j', 'a'},
	"NL": {'n', 'l'},
	"PL": {'p', 'l'},
	"PT": {'p', 't'},
	"RU": {'r', 'u'},
	"UA": {'u', 'k'},
	"US": {'e', 'n'},
}

type acceptLangTag struct {
	base   [2]byte
	region [2]byte
}

func parseAcceptLanguageTags(acceptLang string, out []acceptLangTag) int {
	if acceptLang == "" {
		return 0
	}
	count := 0
	start := 0
	n := len(acceptLang)
	for i := 0; i <= n; i++ {
		if i < n && acceptLang[i] != ',' {
			continue
		}
		token := trimAcceptLangToken(acceptLang[start:i])
		if len(token) > 0 && count < len(out) {
			if tag, ok := parseAcceptLangTag(token); ok {
				out[count] = tag
				count++
			}
		}
		start = i + 1
	}
	return count
}

func trimAcceptLangToken(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	if i := indexByteString(s, ';'); i >= 0 {
		s = s[:i]
		for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
			s = s[:len(s)-1]
		}
	}
	return s
}

func parseAcceptLangTag(token string) (acceptLangTag, bool) {
	var tag acceptLangTag
	if len(token) < 2 {
		return tag, false
	}
	if token[0] >= '0' && token[0] <= '9' {
		return tag, false
	}
	if len(token) > 2 && token[2] == '-' {
		if len(token) < 5 {
			return tag, false
		}
		tag.base[0] = foldASCIILower(token[0])
		tag.base[1] = foldASCIILower(token[1])
		tag.region[0] = foldASCIIUpper(token[3])
		tag.region[1] = foldASCIIUpper(token[4])
		return tag, true
	}
	if len(token) != 2 {
		return tag, false
	}
	tag.base[0] = foldASCIILower(token[0])
	tag.base[1] = foldASCIILower(token[1])
	return tag, true
}

func foldASCIILower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func foldASCIIUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

func indexByteString(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func acceptLangGeoMismatch(acceptLang, geoCountry string) bool {
	if acceptLang == "" || len(geoCountry) != 2 {
		return false
	}
	expected, ok := countryPrimaryLang[geoCountry]
	if !ok {
		return false
	}
	var geo [2]byte
	geo[0] = foldASCIIUpper(geoCountry[0])
	geo[1] = foldASCIIUpper(geoCountry[1])

	var tags [8]acceptLangTag
	n := parseAcceptLanguageTags(acceptLang, tags[:])
	if n == 0 {
		return false
	}
	if tags[0].region[0] != 0 && tags[0].region == geo {
		return false
	}
	for i := 0; i < n; i++ {
		if tags[i].base == expected {
			return false
		}
	}
	return true
}
