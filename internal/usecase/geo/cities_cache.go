package geo

import (
	"strings"
	"sync"
	"unicode"

	"github.com/sday-kenta/backend/internal/entity"
)

// CitiesCache stores supported zones and their display names in memory.
type CitiesCache struct {
	mu    sync.RWMutex
	zones []entity.Zone
}

func (c *CitiesCache) Get() []entity.Zone {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]entity.Zone, len(c.zones))
	copy(res, c.zones)
	return res
}

func (c *CitiesCache) Set(zones []entity.Zone) {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := make([]entity.Zone, len(zones))
	copy(res, zones)
	c.zones = res
}

func (c *CitiesCache) Resolve(value string) (entity.Zone, bool) {
	valueKey := normalizeCityComparable(value)
	if valueKey == "" {
		return entity.Zone{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, zone := range c.zones {
		resolved := normalizeZone(zone)
		if resolved.Name == "" {
			continue
		}
		if normalizeCityComparable(resolved.DisplayName) == valueKey || normalizeCityComparable(resolved.Name) == valueKey {
			return resolved, true
		}
	}
	return entity.Zone{}, false
}

func (c *CitiesCache) CanonicalDisplayName(value string) string {
	zone, ok := c.Resolve(value)
	if !ok {
		return ""
	}
	return zone.DisplayName
}

func (c *CitiesCache) FindMentionedZone(text string) (entity.Zone, bool) {
	normalizedText := normalizeCityComparable(text)
	if normalizedText == "" {
		return entity.Zone{}, false
	}
	haystack := " " + normalizedText + " "

	c.mu.RLock()
	defer c.mu.RUnlock()

	best := entity.Zone{}
	bestLen := 0
	for _, zone := range c.zones {
		resolved := normalizeZone(zone)
		if resolved.Name == "" {
			continue
		}
		needle := normalizeCityComparable(resolved.DisplayName)
		if needle == "" {
			continue
		}
		if strings.Contains(haystack, " "+needle+" ") {
			if l := len([]rune(needle)); l > bestLen {
				best = resolved
				bestLen = l
			}
		}
	}

	if best.Name == "" {
		return entity.Zone{}, false
	}
	return best, true
}

func (c *CitiesCache) DisplayNames(limit int, except string) []string {
	exceptKey := normalizeCityComparable(except)
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]string, 0, len(c.zones))
	seen := make(map[string]struct{}, len(c.zones))
	for _, zone := range c.zones {
		resolved := normalizeZone(zone)
		if resolved.Name == "" {
			continue
		}
		key := normalizeCityComparable(resolved.DisplayName)
		if key == "" {
			continue
		}
		if exceptKey != "" && key == exceptKey {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, resolved.DisplayName)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func normalizeZone(zone entity.Zone) entity.Zone {
	zone.Name = strings.TrimSpace(zone.Name)
	zone.DisplayName = strings.TrimSpace(zone.DisplayName)
	if zone.DisplayName == "" {
		zone.DisplayName = zone.Name
	}
	return zone
}

func normalizeCityComparable(value string) string {
	value = strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "ё", "е")))
	if value == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsSpace(r), r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}

	return strings.Join(strings.Fields(b.String()), " ")
}
