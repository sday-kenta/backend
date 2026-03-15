package geo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/internal/repo"
	"github.com/evrone/go-clean-template/internal/usecase/geo/addressnormalizer"
)

const (
	defaultMaxCityAttempts = 4
	maxSearchResults       = 4
)

type searchAttempt struct {
	city             string
	zoneName         string
	outsideAreaMatch bool
}

type UseCase struct {
	geoRepo         repo.GeoRepo
	webAPI          repo.GeoWebAPI
	citiesCache     *CitiesCache
	maxCityAttempts int
}

func New(geoRepo repo.GeoRepo, webAPI repo.GeoWebAPI, maxCityAttempts int) *UseCase {
	if maxCityAttempts <= 0 {
		maxCityAttempts = defaultMaxCityAttempts
	}
	return &UseCase{
		geoRepo:         geoRepo,
		webAPI:          webAPI,
		citiesCache:     &CitiesCache{},
		maxCityAttempts: maxCityAttempts,
	}
}

func (uc *UseCase) ReloadCities(ctx context.Context) error {
	zones, err := uc.geoRepo.GetZones(ctx)
	if err != nil {
		return fmt.Errorf("GeoUseCase - ReloadCities - uc.geoRepo.GetZones: %w", err)
	}
	uc.citiesCache.Set(zones)
	return nil
}

func (uc *UseCase) ReverseGeocode(ctx context.Context, lat, lon float64) (entity.Address, error) {
	if !validCoordinates(lat, lon) {
		return entity.Address{}, entity.ErrInvalidCoordinates
	}

	if _, err := uc.geoRepo.FindContainingZone(ctx, lat, lon); err != nil {
		if errors.Is(err, entity.ErrZoneNotFound) {
			return entity.Address{}, entity.ErrOutOfAllowedZone
		}
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.geoRepo.FindContainingZone: %w", err)
	}

	cachedAddress, err := uc.geoRepo.GetAddressByCoords(ctx, lat, lon)
	if err == nil {
		return cachedAddress, nil
	}
	if !errors.Is(err, entity.ErrAddressNotFound) {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.geoRepo.GetAddressByCoords: %w", err)
	}

	address, err := uc.webAPI.Reverse(ctx, lat, lon)
	if err != nil {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.webAPI.Reverse: %w", err)
	}

	if err = uc.geoRepo.SaveAddress(ctx, address); err != nil {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.geoRepo.SaveAddress: %w", err)
	}

	return address, nil
}

func (uc *UseCase) Search(ctx context.Context, query, city string) ([]entity.Address, error) {
	trimmedQuery := strings.TrimSpace(query)
	if len([]rune(trimmedQuery)) < 3 {
		return nil, entity.ErrInvalidSearchQuery
	}

	supportedCities := uc.citiesCache.DisplayNames(0, "")
	normalizer := addressnormalizer.New(supportedCities)
	parsed := normalizer.Normalize(trimmedQuery)

	var explicitZone *entity.Zone
	if rawCity := strings.TrimSpace(city); rawCity != "" {
		resolved, ok := uc.citiesCache.Resolve(rawCity)
		if !ok {
			return nil, entity.ErrOutOfAllowedZone
		}
		explicitZone = &resolved
	}

	mentionedZone, hasUnsupportedCity := uc.resolveParsedZone(trimmedQuery, parsed)
	if hasUnsupportedCity {
		return nil, entity.ErrOutOfAllowedZone
	}

	attempts := uc.buildSearchAttempts(explicitZone, mentionedZone)
	attemptResults := make([][]entity.Address, 0, len(attempts))
	sawOutsideArea := false
	for _, attempt := range attempts {
		addresses, err := uc.searchCandidate(ctx, trimmedQuery, attempt)
		if err != nil {
			if errors.Is(err, entity.ErrOutOfAllowedZone) {
				if attempt.outsideAreaMatch {
					sawOutsideArea = true
				}
				continue
			}
			return nil, err
		}
		if len(addresses) == 0 {
			continue
		}
		attemptResults = append(attemptResults, addresses)
	}

	results := mergeAttemptResults(attemptResults, maxSearchResults)
	if len(results) > 0 {
		return results, nil
	}
	if sawOutsideArea {
		return nil, entity.ErrOutOfAllowedZone
	}

	return nil, entity.ErrAddressNotFound
}

func (uc *UseCase) searchCandidate(ctx context.Context, query string, attempt searchAttempt) ([]entity.Address, error) {
	normalizedQuery, normalizedCity := uc.normalizeSearchInput(query, attempt.city)
	addresses, err := uc.webAPI.Search(ctx, normalizedQuery, normalizedCity)
	if err != nil {
		return nil, fmt.Errorf("GeoUseCase - Search - uc.webAPI.Search: %w", err)
	}
	if len(addresses) == 0 {
		return []entity.Address{}, nil
	}

	filtered := make([]entity.Address, 0, len(addresses))
	outside := make([]entity.Address, 0, len(addresses))
	for _, address := range addresses {
		allowed, zoneErr := uc.addressAllowedForAttempt(ctx, address, attempt)
		if zoneErr != nil {
			return nil, zoneErr
		}
		if !allowed {
			outside = append(outside, address)
			continue
		}
		filtered = append(filtered, address)
		_ = uc.geoRepo.SaveAddress(ctx, address)
	}
	if len(filtered) > 0 {
		return filtered, nil
	}
	if len(outside) > 0 && queryExplicitlyTargetsOutsideArea(query, attempt.city, outside) {
		return nil, entity.ErrOutOfAllowedZone
	}
	return []entity.Address{}, nil
}

func (uc *UseCase) addressAllowedForAttempt(ctx context.Context, address entity.Address, attempt searchAttempt) (bool, error) {
	if attempt.zoneName != "" {
		allowed, err := uc.geoRepo.IsPointInZone(ctx, address.Lat, address.Lon, attempt.zoneName)
		if err != nil {
			if errors.Is(err, entity.ErrZoneNotFound) {
				return false, nil
			}
			return false, fmt.Errorf("GeoUseCase - Search - uc.geoRepo.IsPointInZone: %w", err)
		}
		return allowed, nil
	}

	_, err := uc.geoRepo.FindContainingZone(ctx, address.Lat, address.Lon)
	if err != nil {
		if errors.Is(err, entity.ErrZoneNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("GeoUseCase - Search - uc.geoRepo.FindContainingZone: %w", err)
	}
	return true, nil
}

func (uc *UseCase) normalizeSearchInput(query, city string) (string, string) {
	supportedCities := uc.citiesCache.DisplayNames(0, "")
	normalizer := addressnormalizer.New(supportedCities)

	raw := strings.TrimSpace(query)
	city = strings.TrimSpace(city)
	if city != "" {
		if explicitMarkerCity, ok := uc.resolveExplicitLocalityCity(query); ok && normalizeString(explicitMarkerCity) == normalizeString(city) {
			if remainder := stripExplicitLocalityPrefix(query, city); remainder != "" {
				raw = city + ", " + remainder
			}
		}
		rawNorm := normalizeString(raw)
		cityNorm := normalizeString(city)
		if cityNorm != "" && !strings.Contains(rawNorm, cityNorm) {
			raw = city + ", " + raw
		}
	}

	parsed := normalizer.Normalize(raw)
	normalizedQuery := strings.TrimSpace(parsed.SearchFreeForm())
	if normalizedQuery == "" {
		normalizedQuery = strings.TrimSpace(query)
	}

	normalizedCity := city
	if zone, ok := uc.resolveZoneByAddressValue(parsed.City); ok {
		normalizedCity = zone.DisplayName
	} else if zone, ok := uc.resolveZoneByAddressValue(parsed.SettlementName); ok {
		normalizedCity = zone.DisplayName
	}

	return normalizedQuery, normalizedCity
}

func (uc *UseCase) resolveZoneByAddressValue(value string) (entity.Zone, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return entity.Zone{}, false
	}
	return uc.citiesCache.Resolve(value)
}

func (uc *UseCase) resolveParsedZone(query string, parsed addressnormalizer.Address) (*entity.Zone, bool) {
	if explicitLocalityCity, ok := uc.resolveExplicitLocalityCity(query); ok {
		resolved, supported := uc.citiesCache.Resolve(explicitLocalityCity)
		if !supported {
			return nil, true
		}
		return &resolved, false
	}

	if mentioned, ok := uc.citiesCache.FindMentionedZone(query); ok {
		return &mentioned, false
	}

	if resolved, supported := uc.resolveZoneByAddressValue(parsed.City); supported {
		return &resolved, false
	}
	if resolved, supported := uc.resolveZoneByAddressValue(parsed.SettlementName); supported {
		return &resolved, false
	}

	return nil, false
}

func (uc *UseCase) resolveExplicitLocalityCity(query string) (string, bool) {
	normalized := normalizeQueryForCityDetection(query)
	if normalized == "" {
		return "", false
	}

	markers := map[string]struct{}{
		"город": {}, "г": {}, "деревня": {}, "село": {}, "поселок": {}, "пгт": {}, "снт": {},
	}
	tokens := strings.Fields(normalized)
	for i, token := range tokens {
		if _, ok := markers[token]; !ok {
			continue
		}
		words := make([]string, 0, 3)
		for j := i + 1; j < len(tokens) && len(words) < 3; j++ {
			current := tokens[j]
			if current == "" || tokenLooksLikeHouse(current) || tokenLooksLikeStreetType(current) {
				break
			}
			words = append(words, current)
		}
		if len(words) == 0 {
			return "", true
		}
		for size := minInt(len(words), 3); size >= 1; size-- {
			candidate := strings.Join(words[:size], " ")
			return candidate, true
		}
	}

	return "", false
}

func (uc *UseCase) buildSearchAttempts(explicitZone, mentionedZone *entity.Zone) []searchAttempt {
	result := make([]searchAttempt, 0, uc.maxCityAttempts+1)
	seen := make(map[string]struct{}, uc.maxCityAttempts+1)

	appendZone := func(zone entity.Zone, outsideAreaMatch bool) {
		zone = normalizeZone(zone)
		if zone.Name == "" || zone.DisplayName == "" {
			return
		}
		key := normalizeString(zone.Name)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		if uc.maxCityAttempts > 0 && len(result) >= uc.maxCityAttempts {
			return
		}
		seen[key] = struct{}{}
		result = append(result, searchAttempt{city: zone.DisplayName, zoneName: zone.Name, outsideAreaMatch: outsideAreaMatch})
	}

	if mentionedZone != nil {
		appendZone(*mentionedZone, true)
	} else {
		if explicitZone != nil {
			appendZone(*explicitZone, true)
		}
		for _, zone := range uc.citiesCache.Get() {
			appendZone(zone, false)
			if uc.maxCityAttempts > 0 && len(result) >= uc.maxCityAttempts {
				break
			}
		}
	}

	return append(result, searchAttempt{outsideAreaMatch: true})
}

func mergeAttemptResults(groups [][]entity.Address, limit int) []entity.Address {
	if limit <= 0 {
		limit = maxSearchResults
	}

	result := make([]entity.Address, 0, limit)
	seen := make(map[string]struct{}, limit)
	appendUnique := func(address entity.Address) {
		if len(result) >= limit {
			return
		}
		key := normalizeString(address.FullAddress)
		if key == "" {
			key = fmt.Sprintf("%.6f|%.6f", address.Lat, address.Lon)
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, address)
	}

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		appendUnique(group[0])
		if len(result) >= limit {
			return result
		}
	}

	for _, group := range groups {
		for i := 1; i < len(group); i++ {
			appendUnique(group[i])
			if len(result) >= limit {
				return result
			}
		}
	}

	return result
}

func queryExplicitlyTargetsOutsideArea(query, city string, outside []entity.Address) bool {
	queryNorm := normalizeString(query)
	cityNorm := normalizeString(city)
	queryTokens := tokenSet(queryNorm)

	for _, address := range outside {
		addressCity := normalizeString(address.City)
		if cityNorm != "" && addressCity != "" && strings.Contains(addressCity, cityNorm) {
			return true
		}
		if addressCity == "" || !containsTokenSequence(queryNorm, addressCity) {
			continue
		}
		if queryNorm != "" && address.FullAddress != "" && strings.Contains(normalizeString(address.FullAddress), queryNorm) {
			return true
		}
		houseNorm := normalizeString(address.HouseNumber)
		roadTokens := strings.Fields(normalizeString(address.Road))
		if houseNorm != "" {
			if _, ok := queryTokens[houseNorm]; !ok {
				continue
			}
		}
		if len(roadTokens) == 0 || countCommonTokens(queryTokens, roadTokens) > 0 {
			return true
		}
	}

	return false
}

func normalizeQueryForCityDetection(query string) string {
	query = strings.ToLower(strings.ReplaceAll(query, "ё", "е"))
	replacer := strings.NewReplacer(
		".", " ", ",", " ", ";", " ", ":", " ", "(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ", "\t", " ", "\n", " ", "\r", " ",
	)
	query = replacer.Replace(query)
	query = strings.Join(strings.Fields(query), " ")
	return strings.TrimSpace(query)
}

func stripExplicitLocalityPrefix(query, city string) string {
	normalized := normalizeQueryForCityDetection(query)
	if normalized == "" {
		return ""
	}
	tokens := strings.Fields(normalized)
	markers := map[string]struct{}{
		"город": {}, "г": {}, "деревня": {}, "село": {}, "поселок": {}, "пгт": {}, "снт": {},
	}
	cityTokens := strings.Fields(normalizeString(city))
	for i, token := range tokens {
		if _, ok := markers[token]; !ok {
			continue
		}
		start := i + 1
		matched := 0
		for start+matched < len(tokens) && matched < len(cityTokens) {
			if normalizeString(tokens[start+matched]) != cityTokens[matched] {
				break
			}
			matched++
		}
		if matched == len(cityTokens) {
			return strings.Join(tokens[start+matched:], " ")
		}
	}
	return ""
}

func tokenLooksLikeHouse(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func tokenLooksLikeStreetType(token string) bool {
	switch token {
	case "улица", "проспект", "переулок", "проезд", "шоссе", "бульвар", "набережная", "площадь", "тракт", "аллея", "тупик", "линия", "дорога", "микрорайон", "территория", "ул", "пр", "пер", "ш", "наб", "пл":
		return true
	default:
		return false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func containsTokenSequence(haystack, needle string) bool {
	haystack = normalizeString(haystack)
	needle = normalizeString(needle)
	if haystack == "" || needle == "" {
		return false
	}
	return strings.Contains(" "+haystack+" ", " "+needle+" ")
}

func tokenSet(value string) map[string]struct{} {
	tokens := strings.Fields(normalizeString(value))
	result := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		result[token] = struct{}{}
	}
	return result
}

func countCommonTokens(tokens map[string]struct{}, values []string) int {
	count := 0
	for _, value := range values {
		if _, ok := tokens[normalizeString(value)]; ok {
			count++
		}
	}
	return count
}

func validCoordinates(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return false
	}

	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func normalizeString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "ё", "е")
	value = strings.ToLower(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}
