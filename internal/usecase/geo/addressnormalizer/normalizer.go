package addressnormalizer

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Address struct {
	Raw            string
	Clean          string
	Country        string
	Region         string
	District       string
	City           string
	SettlementType string
	SettlementName string
	StreetType     string
	StreetName     string
	HouseType      string
	HouseNumber    string
	Corp           string
	Building       string
	Letter         string
	UnitType       string
	UnitNumber     string
	Extras         []string
	Warnings       []string
}

type Segment struct {
	Raw    string
	Tokens []string
	Used   []bool
}

type span struct {
	start int
	end   int
}

type Normalizer struct {
	supportedCities map[string]struct{}
}

var (
	reMultiSpace      = regexp.MustCompile(`\s+`)
	reLetterDigit     = regexp.MustCompile(`([\p{L}])(\d)`)
	reDigitLetter     = regexp.MustCompile(`(\d)([\p{L}])`)
	rePunctSpace      = regexp.MustCompile(`[;|]+`)
	reTrimComma       = regexp.MustCompile(`\s*,\s*`)
	reGluePrefix      = regexp.MustCompile(`\b(ул|пр|проспект|пер|ш|наб|пл|кв|корп|стр|лит|г|д|с|п)([\p{L}])`)
	reHouseLike       = regexp.MustCompile(`^\d+[\p{L}]?(?:/\d+[\p{L}]?)?$`)
	reHyphenHouseUnit = regexp.MustCompile(`^(\d+)-(\d+)$`)
	reSlashSeparated  = regexp.MustCompile(`(\d+)\s*/\s*(\d+)`)
	reSpacedDash      = regexp.MustCompile(`\s+-\s+`)
)

var unconditionalExpand = map[string]string{
	"ул":      "улица",
	"улиц":    "улица",
	"пр":      "проспект",
	"пр-т":    "проспект",
	"пр-кт":   "проспект",
	"просп":   "проспект",
	"пер":     "переулок",
	"пр-д":    "проезд",
	"прзд":    "проезд",
	"б-р":     "бульвар",
	"бул":     "бульвар",
	"наб":     "набережная",
	"пл":      "площадь",
	"ш":       "шоссе",
	"мкр":     "микрорайон",
	"тер":     "территория",
	"оф":      "офис",
	"пом":     "помещение",
	"кв":      "квартира",
	"ком":     "комната",
	"стр":     "строение",
	"корп":    "корпус",
	"лит":     "литера",
	"р-н":     "район",
	"гор":     "город",
	"пос":     "поселок",
	"посёлок": "поселок",
	"посёл":   "поселок",
	"респ":    "республика",
	"обл":     "область",
	"край":    "край",
	"окр":     "округ",
	"рф":      "россия",
	"вул":     "улица",
	"вулиця":  "улица",
	"буд":     "дом",
}

var countrySynonyms = map[string]bool{
	"россия": true,
}

var stopWords = map[string]bool{
	"дом": true, "корпус": true, "строение": true, "литера": true,
	"квартира": true, "офис": true, "помещение": true, "комната": true,
	"участок": true, "подъезд": true, "этаж": true, "метро": true,
	"район": true, "область": true, "край": true, "республика": true, "округ": true,
	"город": true, "деревня": true, "село": true, "поселок": true, "пгт": true, "снт": true,
	"улица": true, "проспект": true, "переулок": true, "проезд": true,
	"шоссе": true, "бульвар": true, "набережная": true, "площадь": true, "тракт": true,
	"территория": true, "микрорайон": true,
	"без": true, "номера": true,
}

var streetTypes = map[string]string{
	"улица":      "улица",
	"проспект":   "проспект",
	"переулок":   "переулок",
	"проезд":     "проезд",
	"шоссе":      "шоссе",
	"бульвар":    "бульвар",
	"набережная": "набережная",
	"площадь":    "площадь",
	"тракт":      "тракт",
	"аллея":      "аллея",
	"тупик":      "тупик",
	"линия":      "линия",
	"дорога":     "дорога",
	"микрорайон": "микрорайон",
	"территория": "территория",
}

var localityTypes = map[string]string{
	"город":   "город",
	"деревня": "деревня",
	"село":    "село",
	"поселок": "поселок",
	"пгт":     "пгт",
	"снт":     "снт",
}

var buildingTypes = map[string]string{
	"дом":        "дом",
	"владение":   "владение",
	"здание":     "здание",
	"сооружение": "сооружение",
}

var unitTypes = map[string]string{
	"квартира":  "квартира",
	"офис":      "офис",
	"помещение": "помещение",
	"комната":   "комната",
	"участок":   "участок",
}

var extraTypes = map[string]string{
	"метро":   "метро",
	"подъезд": "подъезд",
	"этаж":    "этаж",
}

var monthsOrYear = map[string]bool{
	"января": true, "февраля": true, "марта": true, "апреля": true,
	"мая": true, "июня": true, "июля": true, "августа": true,
	"сентября": true, "октября": true, "ноября": true, "декабря": true,
	"лет": true, "года": true,
}

func New(supportedCities []string) *Normalizer {
	n := &Normalizer{supportedCities: make(map[string]struct{}, len(supportedCities))}
	for _, city := range supportedCities {
		city = strings.TrimSpace(strings.ToLower(strings.ReplaceAll(city, "ё", "е")))
		if city == "" {
			continue
		}
		n.supportedCities[city] = struct{}{}
	}
	return n
}

func (n *Normalizer) Normalize(raw string) Address {
	clean := preprocess(raw)
	segments := splitSegments(clean)
	addr := Address{Raw: raw, Clean: clean}

	for i := range segments {
		n.normalizeSegmentTokens(&segments[i])
	}

	extractCountry(segments, &addr)
	extractExplicitParts(segments, &addr)
	extractLocalityExplicit(segments, &addr)
	extractStreetExplicit(segments, &addr)
	n.inferUsingSupportedCities(segments, &addr)
	n.inferSplitSegments(segments, &addr)
	inferBareUnitSegment(segments, &addr)

	if addr.StreetName == "" {
		n.inferStreetAndHouse(segments, &addr)
	}
	n.inferSplitSegments(segments, &addr)
	inferBareUnitSegment(segments, &addr)

	n.inferLocality(segments, &addr)
	captureLeftovers(segments, &addr)

	addr.Country = normalizeCountry(addr.Country)
	addr.City = titleCase(addr.City)
	addr.District = titleCase(addr.District)
	addr.SettlementName = titleCase(addr.SettlementName)
	addr.StreetName = titleCase(addr.StreetName)
	addr.Region = titleCase(addr.Region)
	addr.Extras = uniqStrings(addr.Extras)
	addr.Warnings = uniqStrings(addr.Warnings)

	return addr
}

func preprocess(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.ReplaceAll(s, "российская федерация", "россия")
	s = strings.ReplaceAll(s, "росс. федерация", "россия")
	s = strings.ReplaceAll(s, "г.", "г ")
	s = strings.ReplaceAll(s, "ул.", "ул ")
	s = strings.ReplaceAll(s, "пер.", "пер ")
	s = strings.ReplaceAll(s, "пр.", "пр ")
	s = strings.ReplaceAll(s, "д.", "д ")
	s = strings.ReplaceAll(s, "кв.", "кв ")
	s = strings.ReplaceAll(s, "корп.", "корп ")
	s = strings.ReplaceAll(s, "стр.", "стр ")
	s = strings.ReplaceAll(s, "лит.", "лит ")
	s = strings.ReplaceAll(s, "р-н", "район")
	s = reSlashSeparated.ReplaceAllString(s, "$1/$2")
	s = reSpacedDash.ReplaceAllString(s, ", ")
	s = reLetterDigit.ReplaceAllString(s, "$1 $2")
	s = reDigitLetter.ReplaceAllString(s, "$1 $2")
	s = reGluePrefix.ReplaceAllString(s, "$1 $2")
	s = rePunctSpace.ReplaceAllString(s, ",")
	repl := strings.NewReplacer("(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ", ":", " ", "\t", " ", "\n", " ", "\r", " ", "№", " ")
	s = repl.Replace(s)
	s = reTrimComma.ReplaceAllString(s, ",")
	s = strings.ReplaceAll(s, ",,", ",")
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = strings.Trim(s, " ,")
	return s
}

func splitSegments(s string) []Segment {
	parts := strings.Split(s, ",")
	out := make([]Segment, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		toks := strings.Fields(p)
		out = append(out, Segment{Raw: p, Tokens: toks, Used: make([]bool, len(toks))})
	}
	if len(out) == 0 {
		return []Segment{{Raw: s, Tokens: strings.Fields(s), Used: make([]bool, len(strings.Fields(s)))}}
	}
	return out
}

func (n *Normalizer) normalizeSegmentTokens(seg *Segment) {
	var out []string
	for i := 0; i < len(seg.Tokens); i++ {
		t := strings.Trim(seg.Tokens[i], " .")
		if t == "" {
			continue
		}
		next := ""
		if i+1 < len(seg.Tokens) {
			next = strings.Trim(seg.Tokens[i+1], " .")
		}
		if split, ok := splitCityGluedToken(t, next, n.supportedCities); ok {
			for _, part := range split {
				if v, ok := unconditionalExpand[part]; ok {
					out = append(out, v)
				} else {
					out = append(out, part)
				}
			}
			continue
		}

		split := splitKnownPrefixToken(t)
		if len(split) == 1 {
			if split2, ok := splitRiskyPrefixWithNextHouse(t, next); ok {
				split = split2
			}
		}
		for _, part := range split {
			if v, ok := unconditionalExpand[part]; ok {
				out = append(out, v)
			} else {
				out = append(out, part)
			}
		}
	}
	seg.Tokens = out
	seg.Used = make([]bool, len(out))
}

func splitKnownPrefixToken(tok string) []string {
	safeNumericPrefixes := []string{"кв", "корп", "стр", "лит", "д", "к", "оф", "пом"}
	for _, p := range safeNumericPrefixes {
		if strings.HasPrefix(tok, p) && len(tok) > len(p) {
			tail := tok[len(p):]
			if regexp.MustCompile(`^\d+[\p{L}]?(?:/\d+[\p{L}]?)?$`).MatchString(tail) {
				return []string{p, tail}
			}
		}
	}

	riskyPrefixes := []string{"улица", "проспект", "переулок", "ул", "пр", "пер", "ш", "наб", "пл"}
	for _, p := range riskyPrefixes {
		if strings.HasPrefix(tok, p) && len(tok) > len(p) {
			tail := tok[len(p):]
			if regexp.MustCompile(`^[\p{L}-]+\d+[\p{L}]?(?:/\d+[\p{L}]?)?$`).MatchString(tail) {
				m := regexp.MustCompile(`^([\p{L}-]+)(\d+[\p{L}]?(?:/\d+[\p{L}]?)?)$`).FindStringSubmatch(tail)
				if len(m) == 3 {
					return []string{p, m[1], m[2]}
				}
			}
		}
	}
	return []string{tok}
}

func splitCityGluedToken(tok, next string, supportedCities map[string]struct{}) ([]string, bool) {
	for city := range supportedCities {
		if strings.Contains(city, " ") {
			continue
		}
		if strings.HasPrefix(tok, city) && len(tok) > len(city) {
			tail := tok[len(city):]
			if m := regexp.MustCompile(`^([\p{L}-]+)(\d+[\p{L}]?(?:/\d+[\p{L}]?)?)$`).FindStringSubmatch(tail); len(m) == 3 {
				return []string{city, m[1], m[2]}, true
			}
			if next != "" && isHouseToken(next) && regexp.MustCompile(`^[\p{L}-]+$`).MatchString(tail) {
				return []string{city, tail}, true
			}
		}
	}
	return nil, false
}

func splitRiskyPrefixWithNextHouse(tok, next string) ([]string, bool) {
	if next == "" || !isHouseToken(next) {
		return nil, false
	}
	if _, ok := streetTypes[tok]; ok {
		return nil, false
	}
	riskyPrefixes := []string{"улица", "проспект", "переулок", "ул", "пр", "пер", "ш", "наб", "пл", "вул", "вулиця"}
	for _, p := range riskyPrefixes {
		if strings.HasPrefix(tok, p) && len(tok) > len(p)+1 {
			tail := tok[len(p):]
			r, _ := utf8.DecodeRuneInString(tail)
			if r == 'ь' || r == 'ъ' {
				return nil, false
			}
			if regexp.MustCompile(`^[\p{L}-]+$`).MatchString(tail) {
				return []string{p, tail}, true
			}
		}
	}
	return nil, false
}

func extractCountry(segs []Segment, addr *Address) {
	for si := range segs {
		s := &segs[si]
		for i, tok := range s.Tokens {
			if s.Used[i] {
				continue
			}
			if countrySynonyms[tok] {
				addr.Country = "Россия"
				s.Used[i] = true
			}
		}
	}
}

func extractExplicitParts(segs []Segment, addr *Address) {
	for si := range segs {
		s := &segs[si]
		for i := 0; i < len(s.Tokens); i++ {
			if s.Used[i] {
				continue
			}
			tok := s.Tokens[i]

			if i == 0 && len(s.Tokens) > 1 && addr.HouseNumber == "" && isHouseToken(tok) {
				addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
				if reHyphenHouseUnit.MatchString(tok) && addr.UnitNumber == "" {
					m := reHyphenHouseUnit.FindStringSubmatch(tok)
					addr.HouseNumber = m[1]
					addr.UnitType = firstNonEmpty(addr.UnitType, "квартира")
					addr.UnitNumber = m[2]
					addr.Warnings = append(addr.Warnings, "дефис в номере интерпретирован как дом-квартира")
				} else {
					addr.HouseNumber = tok
				}
				s.Used[i] = true
				continue
			}

			if tok == "д" {
				if i+1 < len(s.Tokens) && isHouseToken(s.Tokens[i+1]) {
					tok = "дом"
					s.Tokens[i] = tok
				} else if i+1 < len(s.Tokens) {
					tok = "деревня"
					s.Tokens[i] = tok
				}
			}
			if tok == "с" && i+1 < len(s.Tokens) && !isHouseToken(s.Tokens[i+1]) {
				tok = "село"
				s.Tokens[i] = tok
			}
			if tok == "п" && i+1 < len(s.Tokens) && !isHouseToken(s.Tokens[i+1]) {
				tok = "поселок"
				s.Tokens[i] = tok
			}
			if tok == "г" && i+1 < len(s.Tokens) && !isHouseToken(s.Tokens[i+1]) {
				tok = "город"
				s.Tokens[i] = tok
			}
			if tok == "к" && i+1 < len(s.Tokens) && isHouseToken(s.Tokens[i+1]) {
				tok = "корпус"
				s.Tokens[i] = tok
			}
			if tok == "м" && i+1 < len(s.Tokens) && !isHouseToken(s.Tokens[i+1]) {
				tok = "метро"
				s.Tokens[i] = tok
			}

			if bt, ok := buildingTypes[tok]; ok {
				if i+1 < len(s.Tokens) && isHouseToken(s.Tokens[i+1]) {
					addr.HouseType = bt
					addr.HouseNumber = s.Tokens[i+1]
					s.Used[i], s.Used[i+1] = true, true
					i++
					continue
				}
			}
			switch tok {
			case "корпус":
				if i+1 < len(s.Tokens) && isHouseToken(s.Tokens[i+1]) {
					addr.Corp = s.Tokens[i+1]
					s.Used[i], s.Used[i+1] = true, true
					i++
					continue
				}
			case "строение":
				if i+1 < len(s.Tokens) && isHouseToken(s.Tokens[i+1]) {
					addr.Building = s.Tokens[i+1]
					s.Used[i], s.Used[i+1] = true, true
					i++
					continue
				}
			case "литера":
				if i+1 < len(s.Tokens) {
					addr.Letter = strings.ToUpper(s.Tokens[i+1])
					s.Used[i], s.Used[i+1] = true, true
					i++
					continue
				}
			}
			if ut, ok := unitTypes[tok]; ok {
				if i+1 < len(s.Tokens) && isHouseToken(s.Tokens[i+1]) {
					addr.UnitType = ut
					addr.UnitNumber = s.Tokens[i+1]
					s.Used[i], s.Used[i+1] = true, true
					i++
					continue
				}
			}
			if et, ok := extraTypes[tok]; ok {
				if i+1 < len(s.Tokens) {
					addr.Extras = append(addr.Extras, et+" "+titleCase(s.Tokens[i+1]))
					s.Used[i], s.Used[i+1] = true, true
					i++
					continue
				}
			}
		}
	}
}

func extractStreetExplicit(segs []Segment, addr *Address) {
	bestScore := -1
	var bestSeg int
	var bestName span
	var bestType string
	var bestPrefix bool
	var bestMarker int

	for si := range segs {
		s := &segs[si]
		for i, tok := range s.Tokens {
			if s.Used[i] {
				continue
			}
			st, ok := streetTypes[tok]
			if !ok {
				continue
			}
			left := collectStreetNameLeft(s.Tokens, s.Used, i-1)
			right := collectStreetNameRight(s.Tokens, s.Used, i+1)
			ls := scoreStreetCandidate(s.Tokens, s.Used, left, i, false)
			rs := scoreStreetCandidate(s.Tokens, s.Used, right, i, true)
			if rs > bestScore {
				bestScore = rs
				bestSeg = si
				bestName = right
				bestType = st
				bestPrefix = true
				bestMarker = i
			}
			if ls > bestScore {
				bestScore = ls
				bestSeg = si
				bestName = left
				bestType = st
				bestPrefix = false
				bestMarker = i
			}
		}
	}
	if bestScore < 1 {
		return
	}
	s := &segs[bestSeg]
	addr.StreetType = bestType
	addr.StreetName = joinTokens(s.Tokens[bestName.start : bestName.end+1])
	for i := bestName.start; i <= bestName.end; i++ {
		s.Used[i] = true
	}
	if bestMarker >= 0 && bestMarker < len(s.Tokens) {
		s.Used[bestMarker] = true
	}
	inferHouseNearExplicitStreet(s, bestName, bestMarker, bestPrefix, addr)
}

func collectStreetNameLeft(tokens []string, used []bool, i int) span {
	if i < 0 || used[i] {
		return span{-1, -1}
	}
	start := i
	count := 0
	for start >= 0 && !used[start] && count < 5 {
		if isBoundaryToken(tokens[start]) {
			break
		}
		if !isPotentialStreetNameToken(tokens, start, nil) {
			break
		}
		start--
		count++
	}
	start++
	if start > i {
		return span{-1, -1}
	}
	return span{start, i}
}

func collectStreetNameRight(tokens []string, used []bool, i int) span {
	if i >= len(tokens) || used[i] {
		return span{-1, -1}
	}
	end := i
	count := 0
	for end < len(tokens) && !used[end] && count < 5 {
		if isBoundaryToken(tokens[end]) {
			break
		}
		if !isPotentialStreetNameToken(tokens, end, nil) {
			break
		}
		end++
		count++
	}
	end--
	if end < i {
		return span{-1, -1}
	}
	return span{i, end}
}

func scoreStreetCandidate(tokens []string, used []bool, sp span, markerIdx int, prefix bool) int {
	if sp.start < 0 || sp.end < sp.start {
		return -10
	}
	ln := sp.end - sp.start + 1
	score := 0
	if ln >= 1 && ln <= 4 {
		score += 4
	}
	if ln == 5 {
		score += 1
	}
	hasAlpha := false
	for i := sp.start; i <= sp.end; i++ {
		if used[i] || isBoundaryToken(tokens[i]) {
			return -10
		}
		if isAlphaLike(tokens[i]) {
			hasAlpha = true
		}
		if isHouseToken(tokens[i]) && !isNumericStreetNameToken(tokens, i) {
			return -7
		}
	}
	if hasAlpha {
		score += 2
	}
	var after int
	if prefix {
		after = sp.end + 1
	} else {
		after = markerIdx + 1
	}
	if after >= len(tokens) || isHouseOrBuildingMarker(tokens[after]) || isHouseToken(tokens[after]) {
		score += 3
	}
	var before int
	if prefix {
		before = markerIdx - 1
	} else {
		before = sp.start - 1
	}
	if before < 0 || isLocalityMarker(tokens[before]) || countrySynonyms[tokens[before]] {
		score += 2
	}
	return score
}

func (n *Normalizer) inferUsingSupportedCities(segs []Segment, addr *Address) {
	if len(n.supportedCities) == 0 {
		return
	}
	for si := range segs {
		s := &segs[si]
		cityStart, cityEnd, cityName, ok := findSupportedCitySpan(s.Tokens, s.Used, n.supportedCities)
		if !ok {
			continue
		}
		if addr.City == "" {
			addr.City = cityName
			for i := cityStart; i <= cityEnd; i++ {
				s.Used[i] = true
			}
		}
		houseIdx := -1
		for _, idx := range remainingIndexes(*s) {
			if idx >= cityStart && idx <= cityEnd {
				continue
			}
			if isHouseToken(s.Tokens[idx]) {
				houseIdx = idx
				break
			}
		}
		if houseIdx == -1 {
			continue
		}
		streetTokens := make([]string, 0)
		streetType := ""
		for _, i := range remainingIndexes(*s) {
			if (i >= cityStart && i <= cityEnd) || i == houseIdx {
				continue
			}
			tok := s.Tokens[i]
			if _, ok := streetTypes[tok]; ok {
				if streetType == "" {
					streetType = streetTypes[tok]
				}
				continue
			}
			if isSkippableBetweenCityAndHouse(tok) || isAdminOkrug(tok) || tok == "-" || tok == "/" {
				continue
			}
			streetTokens = append(streetTokens, tok)
		}
		if addr.StreetName == "" && len(streetTokens) > 0 {
			addr.StreetName = joinTokens(streetTokens)
			addr.StreetType = firstNonEmpty(streetType, guessStreetTypeFromName(addr.StreetName))
			markStreetTokens(s, streetTokens)
		}
		if addr.HouseNumber == "" {
			addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
			if reHyphenHouseUnit.MatchString(s.Tokens[houseIdx]) && addr.UnitNumber == "" {
				m := reHyphenHouseUnit.FindStringSubmatch(s.Tokens[houseIdx])
				addr.HouseNumber = m[1]
				addr.UnitType = firstNonEmpty(addr.UnitType, "квартира")
				addr.UnitNumber = m[2]
				addr.Warnings = append(addr.Warnings, "дефис в номере интерпретирован как дом-квартира")
			} else {
				addr.HouseNumber = s.Tokens[houseIdx]
			}
			s.Used[houseIdx] = true
		}
		return
	}
}

func findSupportedCitySpan(tokens []string, used []bool, supportedCities map[string]struct{}) (int, int, string, bool) {
	maxWords := 3
	for start := 0; start < len(tokens); start++ {
		if used[start] || isBoundaryToken(tokens[start]) || isHouseToken(tokens[start]) {
			continue
		}
		for words := maxWords; words >= 1; words-- {
			end := start + words - 1
			if end >= len(tokens) {
				continue
			}
			valid := true
			parts := make([]string, 0, words)
			for i := start; i <= end; i++ {
				if used[i] || isBoundaryToken(tokens[i]) || isHouseToken(tokens[i]) {
					valid = false
					break
				}
				parts = append(parts, tokens[i])
			}
			if !valid {
				continue
			}
			candidate := strings.Join(parts, " ")
			if _, ok := supportedCities[candidate]; ok {
				return start, end, candidate, true
			}
		}
	}
	return -1, -1, "", false
}

func isSkippableBetweenCityAndHouse(tok string) bool {
	if tok == "" {
		return true
	}
	if tok == "город" || tok == "дом" || tok == "деревня" || tok == "село" || tok == "поселок" || tok == "район" {
		return true
	}
	if _, ok := streetTypes[tok]; ok {
		return true
	}
	return false
}

func inferHouseAroundStreet(seg *Segment, from int, addr *Address) {
	if addr.HouseNumber != "" {
		return
	}
	for i := from; i < len(seg.Tokens); i++ {
		if seg.Used[i] {
			continue
		}
		tok := seg.Tokens[i]
		if isHouseToken(tok) {
			addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
			if reHyphenHouseUnit.MatchString(tok) && addr.UnitNumber == "" {
				m := reHyphenHouseUnit.FindStringSubmatch(tok)
				addr.HouseNumber = m[1]
				addr.UnitType = firstNonEmpty(addr.UnitType, "квартира")
				addr.UnitNumber = m[2]
				addr.Warnings = append(addr.Warnings, "дефис в номере интерпретирован как дом-квартира")
			} else {
				addr.HouseNumber = tok
			}
			seg.Used[i] = true
			return
		}
	}
}

func inferHouseNearExplicitStreet(seg *Segment, street span, markerIdx int, prefix bool, addr *Address) {
	if addr.HouseNumber != "" {
		return
	}
	if prefix {
		inferHouseAroundStreet(seg, street.end+1, addr)
		if addr.HouseNumber != "" {
			return
		}
		for i := street.start - 1; i >= 0; i-- {
			if seg.Used[i] {
				continue
			}
			if isHouseToken(seg.Tokens[i]) {
				addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
				addr.HouseNumber = seg.Tokens[i]
				seg.Used[i] = true
				return
			}
			if isBoundaryToken(seg.Tokens[i]) {
				break
			}
		}
		return
	}
	for i := street.start - 1; i >= 0; i-- {
		if seg.Used[i] {
			continue
		}
		if isHouseToken(seg.Tokens[i]) {
			addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
			addr.HouseNumber = seg.Tokens[i]
			seg.Used[i] = true
			return
		}
		if isBoundaryToken(seg.Tokens[i]) {
			break
		}
	}
	inferHouseAroundStreet(seg, markerIdx+1, addr)
}

func (n *Normalizer) inferSplitSegments(segs []Segment, addr *Address) {
	if addr.StreetName != "" && addr.HouseNumber == "" {
		for si := range segs {
			if extractHouseFromStandaloneSegment(&segs[si], addr) {
				return
			}
		}
	}

	if addr.StreetName == "" {
		bestScore := -1
		bestSeg := -1
		var bestTokens []string
		bestType := ""
		for si := range segs {
			name, stype, score := n.streetCandidateFromWholeSegment(segs[si], addr)
			if score > bestScore {
				bestScore = score
				bestSeg = si
				bestTokens = name
				bestType = stype
			}
		}
		if bestScore >= 2 && len(bestTokens) > 0 {
			addr.StreetName = joinTokens(bestTokens)
			addr.StreetType = bestType
			markStreetTokens(&segs[bestSeg], bestTokens)
		}
	}

	if addr.StreetName != "" && addr.HouseNumber == "" {
		for si := range segs {
			if extractHouseFromStandaloneSegment(&segs[si], addr) {
				break
			}
		}
	}

	if addr.StreetName == "" && (addr.City != "" || addr.SettlementName != "") {
		for si := range segs {
			name, stype, score := n.streetCandidateFromWholeSegment(segs[si], addr)
			if score >= 2 && len(name) > 0 {
				addr.StreetName = joinTokens(name)
				addr.StreetType = stype
				markStreetTokens(&segs[si], name)
				break
			}
		}
	}
}

func inferBareUnitSegment(segs []Segment, addr *Address) {
	if addr.HouseNumber == "" || addr.UnitNumber != "" {
		return
	}
	for si := range segs {
		s := &segs[si]
		idxs := remainingIndexes(*s)
		if len(idxs) != 1 {
			continue
		}
		tok := s.Tokens[idxs[0]]
		if regexp.MustCompile(`^\d+[\p{L}]?$`).MatchString(tok) {
			addr.UnitType = firstNonEmpty(addr.UnitType, "квартира")
			addr.UnitNumber = tok
			s.Used[idxs[0]] = true
			return
		}
	}
}

func extractHouseFromStandaloneSegment(seg *Segment, addr *Address) bool {
	if addr.HouseNumber != "" {
		return false
	}
	idxs := remainingIndexes(*seg)
	if len(idxs) == 0 {
		return false
	}
	first := idxs[0]
	tok := seg.Tokens[first]
	if !isHouseToken(tok) {
		return false
	}
	addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
	if reHyphenHouseUnit.MatchString(tok) && addr.UnitNumber == "" {
		m := reHyphenHouseUnit.FindStringSubmatch(tok)
		addr.HouseNumber = m[1]
		addr.UnitType = firstNonEmpty(addr.UnitType, "квартира")
		addr.UnitNumber = m[2]
		addr.Warnings = append(addr.Warnings, "дефис в номере интерпретирован как дом-квартира")
	} else {
		addr.HouseNumber = tok
	}
	seg.Used[first] = true
	return true
}

func (n *Normalizer) streetCandidateFromWholeSegment(seg Segment, addr *Address) ([]string, string, int) {
	idxs := remainingIndexes(seg)
	if len(idxs) == 0 {
		return nil, "", -10
	}
	var toks []string
	streetType := ""
	for _, idx := range idxs {
		tok := seg.Tokens[idx]
		if isAdminOkrug(tok) || tok == "-" || tok == "/" {
			continue
		}
		if _, ok := streetTypes[tok]; ok {
			if streetType == "" {
				streetType = streetTypes[tok]
			}
			continue
		}
		if isBoundaryToken(tok) || isHouseToken(tok) || isLocalityMarker(tok) || countrySynonyms[tok] {
			return nil, "", -10
		}
		if _, ok := n.supportedCities[tok]; ok {
			return nil, "", -8
		}
		toks = append(toks, tok)
	}
	if len(toks) == 0 || len(toks) > 4 {
		return nil, "", -10
	}
	return toks, firstNonEmpty(streetType, guessStreetTypeFromName(joinTokens(toks))), 3 + len(toks)
}

func markStreetTokens(seg *Segment, streetTokens []string) {
	if len(streetTokens) == 0 {
		return
	}
	remaining := append([]string(nil), streetTokens...)
	for i, tok := range seg.Tokens {
		if seg.Used[i] {
			continue
		}
		if _, ok := streetTypes[tok]; ok {
			seg.Used[i] = true
			continue
		}
		if len(remaining) > 0 && tok == remaining[0] {
			seg.Used[i] = true
			remaining = remaining[1:]
		}
	}
}

func (n *Normalizer) inferStreetAndHouse(segs []Segment, addr *Address) {
	bestScore := -1
	var bestSeg int
	var bestName span
	var bestHouse string
	var bestType string

	for si := range segs {
		s := &segs[si]
		rem := remainingIndexes(*s)
		if len(rem) == 0 {
			continue
		}
		if isHouseToken(s.Tokens[rem[0]]) && len(rem) > 1 {
			sp := collectStreetNameRightByIndexes(s.Tokens, rem, 1)
			sc := scoreImplicitStreet(s.Tokens, sp, true)
			if sc > bestScore {
				bestScore = sc
				bestSeg = si
				bestName = sp
				bestHouse = s.Tokens[rem[0]]
				bestType = guessStreetTypeFromName(joinTokens(s.Tokens[sp.start : sp.end+1]))
			}
		}
		last := rem[len(rem)-1]
		if isHouseToken(s.Tokens[last]) && len(rem) > 1 {
			sp := collectStreetNameLeftByIndexes(s.Tokens, rem, len(rem)-2)
			sc := scoreImplicitStreet(s.Tokens, sp, false)
			if sc > bestScore {
				bestScore = sc
				bestSeg = si
				bestName = sp
				bestHouse = s.Tokens[last]
				bestType = guessStreetTypeFromName(joinTokens(s.Tokens[sp.start : sp.end+1]))
			}
		}
	}

	if bestScore < 1 {
		return
	}
	s := &segs[bestSeg]
	addr.StreetName = joinTokens(s.Tokens[bestName.start : bestName.end+1])
	addr.StreetType = bestType
	addr.HouseType = firstNonEmpty(addr.HouseType, "дом")
	if addr.HouseNumber == "" {
		if reHyphenHouseUnit.MatchString(bestHouse) && addr.UnitNumber == "" {
			m := reHyphenHouseUnit.FindStringSubmatch(bestHouse)
			addr.HouseNumber = m[1]
			addr.UnitType = firstNonEmpty(addr.UnitType, "квартира")
			addr.UnitNumber = m[2]
			addr.Warnings = append(addr.Warnings, "дефис в номере интерпретирован как дом-квартира")
		} else {
			addr.HouseNumber = bestHouse
		}
	}
	for i := bestName.start; i <= bestName.end; i++ {
		s.Used[i] = true
	}
	for i, tok := range s.Tokens {
		if !s.Used[i] && tok == bestHouse {
			s.Used[i] = true
			break
		}
	}
}

func collectStreetNameRightByIndexes(tokens []string, idxs []int, startPos int) span {
	if startPos >= len(idxs) {
		return span{-1, -1}
	}
	start := idxs[startPos]
	end := start
	count := 0
	for j := startPos; j < len(idxs) && count < 5; j++ {
		i := idxs[j]
		if isBoundaryToken(tokens[i]) || !isPotentialStreetNameToken(tokens, i, nil) {
			break
		}
		end = i
		count++
	}
	return span{start, end}
}

func collectStreetNameLeftByIndexes(tokens []string, idxs []int, pos int) span {
	if pos < 0 || pos >= len(idxs) {
		return span{-1, -1}
	}
	end := idxs[pos]
	start := end
	count := 0
	for j := pos; j >= 0 && count < 5; j-- {
		i := idxs[j]
		if isBoundaryToken(tokens[i]) || !isPotentialStreetNameToken(tokens, i, nil) {
			break
		}
		start = i
		count++
	}
	return span{start, end}
}

func scoreImplicitStreet(tokens []string, sp span, numberFirst bool) int {
	if sp.start < 0 || sp.end < sp.start {
		return -10
	}
	score := 0
	ln := sp.end - sp.start + 1
	if ln >= 1 && ln <= 4 {
		score += 5
	}
	hasAlpha := false
	for i := sp.start; i <= sp.end; i++ {
		if !isPotentialStreetNameToken(tokens, i, nil) {
			return -7
		}
		if isAlphaLike(tokens[i]) {
			hasAlpha = true
		}
	}
	if hasAlpha {
		score += 2
	}
	if numberFirst {
		score += 1
	}
	return score
}

func extractLocalityExplicit(segs []Segment, addr *Address) {
	for si := range segs {
		s := &segs[si]
		for i, tok := range s.Tokens {
			if s.Used[i] {
				continue
			}
			if lt, ok := localityTypes[tok]; ok {
				left := collectNameLeft(s.Tokens, s.Used, i-1)
				right := collectNameRight(s.Tokens, s.Used, i+1)
				ls := scoreLocalityCandidate(s.Tokens, s.Used, left)
				rs := scoreLocalityCandidate(s.Tokens, s.Used, right)
				var chosen span
				if rs >= ls {
					chosen = right
				} else {
					chosen = left
				}
				if chosen.start >= 0 {
					if chosen.end+1 < len(s.Tokens) {
						if _, ok := streetTypes[s.Tokens[chosen.end+1]]; ok && chosen.end > chosen.start {
							chosen.end--
						}
					}
					name := joinTokens(s.Tokens[chosen.start : chosen.end+1])
					if lt == "город" {
						if addr.City == "" {
							addr.City = name
						}
					} else {
						if addr.SettlementName == "" {
							addr.SettlementType = lt
							addr.SettlementName = name
						}
					}
					s.Used[i] = true
					for j := chosen.start; j <= chosen.end; j++ {
						s.Used[j] = true
					}
				}
			}
			if tok == "район" {
				left := collectNameLeft(s.Tokens, s.Used, i-1)
				right := collectNameRight(s.Tokens, s.Used, i+1)
				var chosen span
				if scoreLocalityCandidate(s.Tokens, s.Used, right) >= scoreLocalityCandidate(s.Tokens, s.Used, left) {
					chosen = right
				} else {
					chosen = left
				}
				if chosen.start >= 0 && addr.District == "" {
					addr.District = joinTokens(s.Tokens[chosen.start : chosen.end+1])
					s.Used[i] = true
					for j := chosen.start; j <= chosen.end; j++ {
						s.Used[j] = true
					}
				}
			}
		}
	}
}

func (n *Normalizer) inferLocality(segs []Segment, addr *Address) {
	if addr.City != "" || addr.SettlementName != "" {
		return
	}
	for _, s := range segs {
		if _, _, city, ok := findSupportedCitySpan(s.Tokens, s.Used, n.supportedCities); ok {
			if !isAdminOkrug(city) {
				addr.City = city
				return
			}
		}
	}
	for _, s := range segs {
		idxs := remainingIndexes(s)
		if len(idxs) == 0 {
			continue
		}
		words := make([]string, 0, len(idxs))
		valid := true
		for _, i := range idxs {
			tok := s.Tokens[i]
			if isHouseToken(tok) || isBoundaryToken(tok) || isAdminOkrug(tok) {
				valid = false
				break
			}
			words = append(words, tok)
		}
		joined := joinTokens(words)
		if valid && len(words) >= 1 && len(words) <= 3 && !isAdminOkrug(joined) {
			if addr.City == "" {
				addr.City = joined
				return
			}
		}
	}
}

func captureLeftovers(segs []Segment, addr *Address) {
	cityLower := strings.ToLower(addr.City)
	settlementLower := strings.ToLower(addr.SettlementName)
	for _, s := range segs {
		idxs := remainingIndexes(s)
		if len(idxs) == 0 {
			continue
		}
		var buf []string
		for _, i := range idxs {
			tok := s.Tokens[i]
			if isAdminOkrug(tok) {
				addr.District = strings.ToUpper(tok)
				continue
			}
			if tok == "д" || tok == "г" || tok == "п" || tok == "с" || tok == "к" || tok == "м" {
				continue
			}
			if tok == cityLower || tok == settlementLower {
				continue
			}
			buf = append(buf, tok)
		}
		if len(buf) > 0 {
			joined := titleCase(joinTokens(buf))
			if strings.EqualFold(joined, addr.City) || strings.EqualFold(joined, addr.SettlementName) {
				continue
			}
			addr.Extras = append(addr.Extras, joined)
		}
	}
}

func collectNameLeft(tokens []string, used []bool, i int) span {
	if i < 0 || used[i] || isBoundaryToken(tokens[i]) {
		return span{-1, -1}
	}
	start := i
	count := 0
	for start >= 0 && !used[start] && count < 4 {
		if isBoundaryToken(tokens[start]) || isHouseToken(tokens[start]) {
			break
		}
		start--
		count++
	}
	start++
	if start > i {
		return span{-1, -1}
	}
	return span{start, i}
}

func collectNameRight(tokens []string, used []bool, i int) span {
	if i >= len(tokens) || used[i] || isBoundaryToken(tokens[i]) {
		return span{-1, -1}
	}
	end := i
	count := 0
	for end < len(tokens) && !used[end] && count < 4 {
		if isBoundaryToken(tokens[end]) || isHouseToken(tokens[end]) {
			break
		}
		end++
		count++
	}
	end--
	if end < i {
		return span{-1, -1}
	}
	return span{i, end}
}

func scoreLocalityCandidate(tokens []string, used []bool, sp span) int {
	if sp.start < 0 || sp.end < sp.start {
		return -10
	}
	score := 0
	ln := sp.end - sp.start + 1
	if ln >= 1 && ln <= 3 {
		score += 4
	}
	for i := sp.start; i <= sp.end; i++ {
		if used[i] || isBoundaryToken(tokens[i]) || isHouseToken(tokens[i]) {
			return -10
		}
	}
	score += 1
	return score
}

func remainingIndexes(seg Segment) []int {
	out := make([]int, 0, len(seg.Tokens))
	for i := range seg.Tokens {
		if !seg.Used[i] {
			out = append(out, i)
		}
	}
	return out
}

func isBoundaryToken(tok string) bool {
	if tok == "" {
		return true
	}
	if stopWords[tok] {
		return true
	}
	if tok == "д" || tok == "г" || tok == "с" || tok == "п" || tok == "к" || tok == "м" || tok == "-" || tok == "/" {
		return true
	}
	return false
}

func isLocalityMarker(tok string) bool {
	_, ok := localityTypes[tok]
	return ok || tok == "район" || tok == "область" || tok == "край" || tok == "республика" || tok == "округ"
}

func isAdminOkrug(tok string) bool {
	s := strings.ToLower(strings.TrimSpace(tok))
	return s == "цао" || s == "юао" || s == "сзао" || s == "вао" || s == "юзао" || s == "сао" || s == "ювао" || s == "свао" || s == "зао"
}

func isHouseOrBuildingMarker(tok string) bool {
	if _, ok := buildingTypes[tok]; ok {
		return true
	}
	if tok == "корпус" || tok == "строение" || tok == "литера" {
		return true
	}
	if _, ok := unitTypes[tok]; ok {
		return true
	}
	return false
}

func isHouseToken(tok string) bool {
	if reHouseLike.MatchString(tok) {
		return true
	}
	if reHyphenHouseUnit.MatchString(tok) {
		return true
	}
	return false
}

func isPotentialStreetNameToken(tokens []string, idx int, supportedCities map[string]struct{}) bool {
	tok := tokens[idx]
	if tok == "" {
		return false
	}
	if isBoundaryToken(tok) || isAdminOkrug(tok) {
		return false
	}
	if supportedCities != nil {
		if _, ok := supportedCities[tok]; ok {
			return false
		}
	}
	if isHouseToken(tok) {
		return isNumericStreetNameToken(tokens, idx)
	}
	return true
}

func isNumericStreetNameToken(tokens []string, idx int) bool {
	tok := tokens[idx]
	if !regexp.MustCompile(`^\d+$`).MatchString(tok) {
		return false
	}
	if idx+1 < len(tokens) && monthsOrYear[tokens[idx+1]] {
		return true
	}
	if idx-1 >= 0 && monthsOrYear[tokens[idx-1]] {
		return true
	}
	return false
}

func isAlphaLike(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func joinTokens(tokens []string) string {
	return strings.TrimSpace(strings.Join(tokens, " "))
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	for i, p := range parts {
		switch strings.ToLower(p) {
		case "снт":
			parts[i] = "СНТ"
			continue
		case "пгт":
			parts[i] = "ПГТ"
			continue
		case "цао", "юао", "сзао", "вао", "юзао", "сао", "ювао", "свао", "зао":
			parts[i] = strings.ToUpper(p)
			continue
		}
		r, size := utf8.DecodeRuneInString(p)
		if r == utf8.RuneError {
			continue
		}
		parts[i] = string(unicode.ToUpper(r)) + strings.ToLower(p[size:])
	}
	return strings.Join(parts, " ")
}

func normalizeCountry(s string) string {
	if s == "" {
		return ""
	}
	return "Россия"
}

func guessStreetTypeFromName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(lower, " проспект"), strings.HasPrefix(lower, "проспект "):
		return "проспект"
	case strings.Contains(lower, " переулок"), strings.HasPrefix(lower, "переулок "):
		return "переулок"
	case strings.Contains(lower, " шоссе"), strings.HasPrefix(lower, "шоссе "):
		return "шоссе"
	case strings.Contains(lower, " набережная"), strings.HasPrefix(lower, "набережная "):
		return "набережная"
	case strings.Contains(lower, " бульвар"), strings.HasPrefix(lower, "бульвар "):
		return "бульвар"
	case strings.Contains(lower, " площадь"), strings.HasPrefix(lower, "площадь "):
		return "площадь"
	default:
		return "улица"
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func uniqStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func (a Address) Canonical() string {
	var parts []string
	if a.Country != "" {
		parts = append(parts, a.Country)
	}
	if a.Region != "" {
		parts = append(parts, a.Region)
	}
	if a.District != "" {
		parts = append(parts, "район "+a.District)
	}
	if a.City != "" {
		parts = append(parts, a.City)
	} else if a.SettlementName != "" {
		parts = append(parts, titleCase(a.SettlementType)+" "+a.SettlementName)
	}
	if a.StreetName != "" {
		parts = append(parts, titleCase(a.StreetType)+" "+a.StreetName)
	}
	if a.HouseNumber != "" {
		houseType := firstNonEmpty(a.HouseType, "дом")
		parts = append(parts, houseType+" "+a.HouseNumber)
	}
	if a.Corp != "" {
		parts = append(parts, "корпус "+a.Corp)
	}
	if a.Building != "" {
		parts = append(parts, "строение "+a.Building)
	}
	if a.Letter != "" {
		parts = append(parts, "литера "+a.Letter)
	}
	if a.UnitNumber != "" {
		unitType := firstNonEmpty(a.UnitType, "квартира")
		parts = append(parts, unitType+" "+a.UnitNumber)
	}
	return strings.Join(parts, ", ")
}

func (a Address) SearchFreeForm() string {
	var parts []string
	if a.City != "" {
		parts = append(parts, a.City)
	} else if a.SettlementName != "" {
		parts = append(parts, a.SettlementName)
	}
	if a.StreetName != "" {
		parts = append(parts, a.StreetName)
	}
	if a.HouseNumber != "" {
		hn := a.HouseNumber
		if a.Corp != "" {
			hn += " корпус " + a.Corp
		}
		if a.Building != "" {
			hn += " строение " + a.Building
		}
		parts = append(parts, hn)
	}
	if len(parts) == 0 {
		return a.Clean
	}
	return strings.Join(parts, ", ")
}
