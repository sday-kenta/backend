package v1

import "strings"

func buildObjectURL(baseURL, key string) string {
	if strings.TrimSpace(baseURL) == "" {
		return key
	}

	return strings.TrimRight(baseURL, "/") + "/" + key
}

func objectKeyFromStoredURL(baseURL, stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}

	if !strings.Contains(stored, "://") {
		return stored
	}

	if strings.TrimSpace(baseURL) == "" {
		return ""
	}

	prefix := strings.TrimRight(baseURL, "/") + "/"
	if strings.HasPrefix(stored, prefix) {
		return strings.TrimPrefix(stored, prefix)
	}

	return ""
}
