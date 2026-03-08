package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/evrone/go-clean-template/internal/entity"
)

const (
	defaultNominatimBaseURL        = "https://nominatim.openstreetmap.org"
	defaultNominatimTimeout        = 10 * time.Second
	defaultNominatimUserAgent      = "sday-kenta/1.0"
	defaultNominatimAcceptLanguage = "ru"
	defaultNominatimCountryCodes   = "ru"
	defaultNominatimSearchLimit    = 8
	defaultNominatimReverseZoom    = 18
	defaultPublicAPIMinInterval    = time.Second
	searchCacheTTL                 = 5 * time.Minute
	defaultSearchCity              = "Самара"
)

var (
	houseNumberAtEndRe   = regexp.MustCompile(`(?i)^(.+?)\s+(\d+[\p{L}\p{N}\-/]*)$`)
	houseNumberAtStartRe = regexp.MustCompile(`(?i)^(\d+[\p{L}\p{N}\-/]*)\s+(.+)$`)
	streetPrefixRe       = regexp.MustCompile(`(?i)\b(г\.?|город|ул\.?|улица|пр[- ]?т\.?|проспект|пер\.?|переулок|бул\.?|бульвар|наб\.?|набережная|ш\.?|шоссе|пл\.?|площадь|пр\.?|проезд)\b`)
	spaceRe              = regexp.MustCompile(`\s+`)

	commonStreetTypeExpansions = []string{
		"проспект",
		"улица",
		"шоссе",
	}
)

type Config struct {
	BaseURL        string
	UserAgent      string
	Email          string
	AcceptLanguage string
	CountryCodes   string
	SearchLimit    int
	ReverseZoom    int
	Timeout        time.Duration
}

type NominatimRepo struct {
	baseURL        string
	userAgent      string
	email          string
	acceptLanguage string
	countryCodes   string
	searchLimit    int
	reverseZoom    int
	httpClient     *http.Client

	throttleMu    sync.Mutex
	lastRequestAt time.Time

	searchCacheMu sync.RWMutex
	searchCache   map[string]cachedSearchResult
}

type cachedSearchResult struct {
	results   []entity.Address
	expiresAt time.Time
}

type parsedAddressQuery struct {
	Original      string
	Normalized    string
	City          string
	Street        string
	HouseNumber   string
	FreeFormQuery string
}

type scoredAddress struct {
	address entity.Address
	score   int
}

func NewNominatimRepo(cfg Config) *NominatimRepo {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaultNominatimBaseURL
	}
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = defaultNominatimUserAgent
	}
	if strings.TrimSpace(cfg.AcceptLanguage) == "" {
		cfg.AcceptLanguage = defaultNominatimAcceptLanguage
	}
	if strings.TrimSpace(cfg.CountryCodes) == "" {
		cfg.CountryCodes = defaultNominatimCountryCodes
	}
	if cfg.SearchLimit <= 0 {
		cfg.SearchLimit = defaultNominatimSearchLimit
	}
	if cfg.SearchLimit > 40 {
		cfg.SearchLimit = 40
	}
	if cfg.ReverseZoom <= 0 {
		cfg.ReverseZoom = defaultNominatimReverseZoom
	}
	if cfg.ReverseZoom > 18 {
		cfg.ReverseZoom = 18
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultNominatimTimeout
	}

	return &NominatimRepo{
		baseURL:        strings.TrimRight(cfg.BaseURL, "/"),
		userAgent:      cfg.UserAgent,
		email:          strings.TrimSpace(cfg.Email),
		acceptLanguage: cfg.AcceptLanguage,
		countryCodes:   strings.ToLower(strings.TrimSpace(cfg.CountryCodes)),
		searchLimit:    cfg.SearchLimit,
		reverseZoom:    cfg.ReverseZoom,
		httpClient:     &http.Client{Timeout: cfg.Timeout},
		searchCache:    make(map[string]cachedSearchResult),
	}
}

func (r *NominatimRepo) Reverse(ctx context.Context, lat, lon float64) (entity.Address, error) {
	values := url.Values{}
	values.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	values.Set("lon", strconv.FormatFloat(lon, 'f', -1, 64))
	values.Set("format", "jsonv2")
	values.Set("addressdetails", "1")
	values.Set("zoom", strconv.Itoa(r.reverseZoom))
	values.Set("layer", "address")
	values.Set("accept-language", r.acceptLanguage)
	if r.email != "" {
		values.Set("email", r.email)
	}

	var place NominatimPlace
	if err := r.doJSON(ctx, http.MethodGet, r.baseURL+"/reverse", values, &place); err != nil {
		return entity.Address{}, err
	}

	if place.Error != "" {
		return entity.Address{}, entity.ErrAddressNotFound
	}

	address, err := mapNominatimPlace(place)
	if err != nil {
		return entity.Address{}, err
	}

	return address, nil
}

func (r *NominatimRepo) Search(ctx context.Context, query string) ([]entity.Address, error) {
	parsed := parseAddressQuery(query)
	cacheKey := parsed.Normalized
	if cached, ok := r.getCachedSearch(cacheKey); ok {
		return cached, nil
	}

	collected := make([]entity.Address, 0, r.searchLimit)
	seen := make(map[string]struct{})

	freeFormQueries := r.buildFreeFormSearchQueries(parsed)
	for _, freeFormValues := range freeFormQueries {
		addresses, err := r.searchRequest(ctx, freeFormValues)
		if err != nil {
			return nil, err
		}
		collected = appendUniqueAddresses(collected, addresses, seen)

		matched := rankAndFilterSearchResults(parsed, collected, r.searchLimit)
		if len(matched) > 0 {
			break
		}
	}

	addresses := rankAndFilterSearchResults(parsed, collected, r.searchLimit)
	addresses = dedupeAddresses(addresses, r.searchLimit)
	if len(addresses) > 0 {
		r.setCachedSearch(cacheKey, addresses)
	}

	return addresses, nil
}

func (r *NominatimRepo) searchRequest(ctx context.Context, values url.Values) ([]entity.Address, error) {
	var places []NominatimPlace
	if err := r.doJSON(ctx, http.MethodGet, r.baseURL+"/search", values, &places); err != nil {
		return nil, err
	}

	addresses := make([]entity.Address, 0, len(places))
	for _, place := range places {
		address, err := mapNominatimPlace(place)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

func (r *NominatimRepo) buildStructuredSearchQueries(parsed parsedAddressQuery) []url.Values {
	if parsed.Street == "" {
		return nil
	}

	streetVariants := []string{parsed.Street}
	if parsed.HouseNumber != "" {
		streetVariants = append(streetVariants, parsed.Street+" "+parsed.HouseNumber, parsed.HouseNumber+" "+parsed.Street)
	}

	queries := make([]url.Values, 0, len(streetVariants))
	seen := make(map[string]struct{})
	for _, street := range streetVariants {
		street = strings.TrimSpace(street)
		if street == "" {
			continue
		}
		key := normalizeString(street)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		values := r.baseSearchValues()
		values.Set("street", street)
		values.Set("city", parsed.City)
		values.Set("country", "Россия")
		values.Set("layer", "address")
		queries = append(queries, values)
	}

	return queries
}

func (r *NominatimRepo) buildFreeFormSearchQueries(parsed parsedAddressQuery) []url.Values {
	queries := make([]url.Values, 0, 4)
	variants := make([]string, 0, 6)

	original := strings.TrimSpace(parsed.Original)
	if original != "" {
		variants = append(variants, original)
	}

	cityForBias := parsed.City
	if cityForBias == "" {
		cityForBias = defaultSearchCity
	}

	if parsed.Street != "" {
		if hasExplicitStreetType(parsed.Street) {
			variant := buildFreeFormAddress(cityForBias, parsed.Street, parsed.HouseNumber)
			if variant != "" {
				variants = append(variants, variant)
			}
		} else {
			expanded := expandStreetVariants(parsed.Street)
			for i, expandedStreet := range expanded {
				if i >= 2 {
					break
				}
				variant := buildFreeFormAddress(cityForBias, expandedStreet, parsed.HouseNumber)
				if variant != "" {
					variants = append(variants, variant)
				}
			}
		}
	}

	formatted := buildFreeFormAddress(parsed.City, parsed.Street, parsed.HouseNumber)
	if formatted != "" {
		variants = append(variants, formatted)
	}

	if parsed.City == "" && strings.TrimSpace(parsed.Original) != "" {
		variants = append(variants, strings.TrimSpace(parsed.Original+", "+defaultSearchCity))
	}

	seen := make(map[string]struct{})
	for _, variant := range variants {
		variant = strings.TrimSpace(variant)
		if variant == "" {
			continue
		}
		key := normalizeString(variant)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		values := r.baseSearchValues()
		values.Set("q", variant)
		queries = append(queries, values)
		if len(queries) >= 4 {
			break
		}
	}

	return queries
}

func (r *NominatimRepo) baseSearchValues() url.Values {
	values := url.Values{}
	values.Set("format", "jsonv2")
	values.Set("addressdetails", "1")
	values.Set("limit", strconv.Itoa(r.searchLimit))
	values.Set("dedupe", "1")
	values.Set("accept-language", r.acceptLanguage)
	if r.countryCodes != "" {
		values.Set("countrycodes", r.countryCodes)
	}
	if r.email != "" {
		values.Set("email", r.email)
	}
	return values
}

func (r *NominatimRepo) doJSON(ctx context.Context, method, endpoint string, query url.Values, dest interface{}) error {
	if err := r.waitTurn(ctx); err != nil {
		return fmt.Errorf("NominatimRepo - waitTurn: %w", err)
	}

	requestURL := endpoint + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return fmt.Errorf("NominatimRepo - http.NewRequestWithContext: %w", err)
	}

	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "application/json")
	if r.acceptLanguage != "" {
		req.Header.Set("Accept-Language", r.acceptLanguage)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("NominatimRepo - httpClient.Do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("NominatimRepo - io.ReadAll: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NominatimRepo - unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err = json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("NominatimRepo - json.Unmarshal: %w", err)
	}

	return nil
}

func (r *NominatimRepo) waitTurn(ctx context.Context) error {
	r.throttleMu.Lock()
	defer r.throttleMu.Unlock()

	remaining := time.Until(r.lastRequestAt.Add(defaultPublicAPIMinInterval))
	if remaining > 0 {
		timer := time.NewTimer(remaining)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	r.lastRequestAt = time.Now()
	return nil
}

func (r *NominatimRepo) getCachedSearch(key string) ([]entity.Address, bool) {
	r.searchCacheMu.RLock()
	cached, ok := r.searchCache[key]
	r.searchCacheMu.RUnlock()
	if !ok || time.Now().After(cached.expiresAt) {
		return nil, false
	}

	results := make([]entity.Address, len(cached.results))
	copy(results, cached.results)
	return results, true
}

func (r *NominatimRepo) setCachedSearch(key string, addresses []entity.Address) {
	results := make([]entity.Address, len(addresses))
	copy(results, addresses)

	r.searchCacheMu.Lock()
	r.searchCache[key] = cachedSearchResult{
		results:   results,
		expiresAt: time.Now().Add(searchCacheTTL),
	}
	r.searchCacheMu.Unlock()
}

func mapNominatimPlace(place NominatimPlace) (entity.Address, error) {
	lat, err := strconv.ParseFloat(place.Lat, 64)
	if err != nil {
		return entity.Address{}, fmt.Errorf("NominatimRepo - ParseFloat lat: %w", err)
	}
	lon, err := strconv.ParseFloat(place.Lon, 64)
	if err != nil {
		return entity.Address{}, fmt.Errorf("NominatimRepo - ParseFloat lon: %w", err)
	}

	fullAddress := strings.TrimSpace(place.DisplayName)
	if fullAddress == "" {
		return entity.Address{}, entity.ErrAddressNotFound
	}

	return entity.Address{
		Lat:         lat,
		Lon:         lon,
		City:        place.Address.GetCity(),
		Road:        strings.TrimSpace(normalizeStreetForDisplay(place.Address.GetRoad())),
		HouseNumber: strings.TrimSpace(place.Address.HouseNumber),
		FullAddress: fullAddress,
	}, nil
}

func parseAddressQuery(query string) parsedAddressQuery {
	normalized := normalizeQuery(query)
	parsed := parsedAddressQuery{
		Original:   strings.TrimSpace(query),
		Normalized: normalized,
	}

	working := normalized
	for _, marker := range []string{"самара", "г самара", "город самара"} {
		if strings.Contains(working, marker) {
			parsed.City = defaultSearchCity
		}
		working = strings.ReplaceAll(working, marker, " ")
	}
	working = normalizeQuery(working)

	if matches := houseNumberAtEndRe.FindStringSubmatch(working); len(matches) == 3 {
		parsed.Street = titleCaseRussian(matches[1])
		parsed.HouseNumber = strings.TrimSpace(matches[2])
	} else if matches := houseNumberAtStartRe.FindStringSubmatch(working); len(matches) == 3 {
		parsed.HouseNumber = strings.TrimSpace(matches[1])
		parsed.Street = titleCaseRussian(matches[2])
	} else {
		parsed.Street = titleCaseRussian(working)
	}

	parts := make([]string, 0, 3)
	if parsed.Street != "" {
		parts = append(parts, parsed.Street)
	}
	if parsed.HouseNumber != "" {
		parts = append(parts, parsed.HouseNumber)
	}
	parts = append(parts, parsed.City)
	parsed.FreeFormQuery = strings.Join(parts, ", ")

	return parsed
}

func rankAndFilterSearchResults(parsed parsedAddressQuery, addresses []entity.Address, limit int) []entity.Address {
	if len(addresses) == 0 {
		return []entity.Address{}
	}

	wantedStreetTokens := significantTokens(parsed.Street)
	wantedHouse := normalizeHouseNumber(parsed.HouseNumber)
	wantedCity := normalizeString(parsed.City)

	scored := make([]scoredAddress, 0, len(addresses))
	for _, address := range addresses {
		if !matchesSearchQuery(parsed, address) {
			continue
		}

		score := 0
		roadNorm := normalizeString(address.Road)
		fullNorm := normalizeString(address.FullAddress)
		cityNorm := normalizeString(address.City)
		houseNorm := normalizeHouseNumber(address.HouseNumber)

		if wantedCity != "" && (strings.Contains(cityNorm, wantedCity) || strings.Contains(fullNorm, wantedCity)) {
			score += 40
		}

		matchCount := 0
		for _, token := range wantedStreetTokens {
			if containsWholeToken(roadNorm, token) {
				matchCount += 2
				continue
			}
			if containsWholeToken(fullNorm, token) {
				matchCount++
			}
		}
		if matchCount > 0 {
			score += 20 * matchCount
			if len(wantedStreetTokens) > 0 && allStreetTokensPresent(address, wantedStreetTokens) {
				score += 30
			}
		}

		if wantedHouse != "" {
			switch {
			case houseNorm == wantedHouse:
				score += 60
			case relaxedHouseMatch(houseNorm, wantedHouse):
				score += 20
			}
			if strings.HasPrefix(fullNorm, wantedHouse+" ") || fullNorm == wantedHouse {
				score += 25
			}
		}

		score += completenessScore(address)
		scored = append(scored, scoredAddress{address: address, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].address.FullAddress < scored[j].address.FullAddress
		}
		return scored[i].score > scored[j].score
	})

	results := make([]entity.Address, 0, min(limit, len(scored)))
	for _, item := range scored {
		results = append(results, item.address)
		if len(results) >= limit {
			break
		}
	}
	return results
}

func matchesSearchQuery(parsed parsedAddressQuery, address entity.Address) bool {
	cityNorm := normalizeString(address.City)
	roadNorm := normalizeString(address.Road)
	fullNorm := normalizeString(address.FullAddress)
	wantedCity := normalizeString(parsed.City)
	wantedHouse := normalizeHouseNumber(parsed.HouseNumber)
	wantedStreetTokens := significantTokens(parsed.Street)

	if wantedCity != "" && !(strings.Contains(cityNorm, wantedCity) || strings.Contains(fullNorm, wantedCity)) {
		return false
	}

	if len(wantedStreetTokens) > 0 {
		if !allTokensPresentInText(roadNorm, wantedStreetTokens) && !allTokensPresentInText(fullNorm, wantedStreetTokens) {
			return false
		}
	}

	if wantedHouse != "" {
		candidateHouse := normalizeHouseNumber(address.HouseNumber)
		if candidateHouse == "" {
			return false
		}
		if candidateHouse != wantedHouse && !relaxedHouseMatch(candidateHouse, wantedHouse) {
			return false
		}
	}

	return true
}

func allStreetTokensPresent(address entity.Address, tokens []string) bool {
	roadNorm := normalizeString(address.Road)
	fullNorm := normalizeString(address.FullAddress)
	return allTokensPresentInText(roadNorm, tokens) || allTokensPresentInText(fullNorm, tokens)
}

func allTokensPresentInText(text string, tokens []string) bool {
	if text == "" || len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if !containsWholeToken(text, token) {
			return false
		}
	}
	return true
}

func containsWholeToken(text, token string) bool {
	if text == "" || token == "" {
		return false
	}
	for _, part := range strings.Fields(text) {
		if part == token {
			return true
		}
	}
	return false
}

func relaxedHouseMatch(candidate, wanted string) bool {
	if candidate == "" || wanted == "" {
		return false
	}
	if candidate == wanted {
		return true
	}
	candidateBase := leadingDigits(candidate)
	wantedBase := leadingDigits(wanted)
	if candidateBase == "" || wantedBase == "" {
		return false
	}
	return candidateBase == wantedBase && len(wantedBase) >= 2
}

func leadingDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if !unicode.IsDigit(r) {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func appendUniqueAddresses(dst, src []entity.Address, seen map[string]struct{}) []entity.Address {
	for _, address := range src {
		key := normalizeString(address.FullAddress)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, address)
	}
	return dst
}

func dedupeAddresses(addresses []entity.Address, limit int) []entity.Address {
	if len(addresses) == 0 {
		return []entity.Address{}
	}

	result := make([]entity.Address, 0, min(limit, len(addresses)))
	seen := make(map[string]struct{})
	for _, address := range addresses {
		key := normalizeString(address.City) + "|" + normalizeString(address.Road) + "|" + normalizeHouseNumber(address.HouseNumber)
		if key == "||" {
			key = normalizeString(address.FullAddress)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, address)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func buildFreeFormAddress(city, street, house string) string {
	parts := make([]string, 0, 3)
	if city = strings.TrimSpace(city); city != "" {
		parts = append(parts, city)
	}
	streetPart := strings.TrimSpace(street)
	if house = strings.TrimSpace(house); house != "" {
		if streetPart != "" {
			streetPart = streetPart + " " + house
		} else {
			streetPart = house
		}
	}
	if streetPart != "" {
		parts = append(parts, streetPart)
	}
	return strings.Join(parts, ", ")
}

func hasExplicitStreetType(street string) bool {
	norm := normalizeString(street)
	if norm == "" {
		return false
	}
	return streetPrefixRe.MatchString(norm)
}

func expandStreetVariants(street string) []string {
	norm := titleCaseRussian(street)
	if norm == "" {
		return nil
	}
	if hasExplicitStreetType(norm) {
		return []string{norm}
	}

	variants := make([]string, 0, len(commonStreetTypeExpansions))
	seen := make(map[string]struct{}, len(commonStreetTypeExpansions))
	for _, prefix := range commonStreetTypeExpansions {
		candidate := strings.TrimSpace(prefix + " " + norm)
		key := normalizeString(candidate)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		variants = append(variants, candidate)
	}
	return variants
}

func normalizeQuery(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "ё", "е")
	value = strings.ReplaceAll(value, ",", " ")
	value = strings.ReplaceAll(value, ";", " ")
	value = strings.ReplaceAll(value, ".", " ")
	return strings.TrimSpace(spaceRe.ReplaceAllString(value, " "))
}

func normalizeString(value string) string {
	value = normalizeQuery(value)
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r == '-' || r == '/' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(spaceRe.ReplaceAllString(b.String(), " "))
}

func significantTokens(value string) []string {
	norm := normalizeString(value)
	if norm == "" {
		return nil
	}
	parts := strings.Fields(norm)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len([]rune(part)) < 2 {
			continue
		}
		result = append(result, part)
	}
	return result
}

func normalizeHouseNumber(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, " ", "")))
}

func completenessScore(address entity.Address) int {
	score := 0
	if strings.TrimSpace(address.Road) != "" {
		score += 5
	}
	if strings.TrimSpace(address.HouseNumber) != "" {
		score += 5
	}
	if strings.TrimSpace(address.City) != "" {
		score += 5
	}
	return score
}

func normalizeStreetForDisplay(value string) string {
	value = strings.TrimSpace(spaceRe.ReplaceAllString(value, " "))
	if value == "" {
		return ""
	}
	return value
}

func titleCaseRussian(value string) string {
	value = normalizeString(value)
	if value == "" {
		return ""
	}
	words := strings.Fields(value)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
