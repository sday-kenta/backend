package v1

import "testing"

func TestBuildObjectURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		key     string
		want    string
	}{
		{
			name:    "empty base keeps key",
			baseURL: "",
			key:     "users/avatar.png",
			want:    "users/avatar.png",
		},
		{
			name:    "joins base and key",
			baseURL: "http://localhost:9000/public",
			key:     "users/avatar.png",
			want:    "http://localhost:9000/public/users/avatar.png",
		},
		{
			name:    "trims trailing slash",
			baseURL: "http://localhost:9000/public/",
			key:     "incidents/1/photo.jpg",
			want:    "http://localhost:9000/public/incidents/1/photo.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildObjectURL(tt.baseURL, tt.key)
			if got != tt.want {
				t.Fatalf("buildObjectURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestObjectKeyFromStoredURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		stored  string
		want    string
	}{
		{
			name:    "empty stored returns empty",
			baseURL: "http://localhost:9000/public",
			stored:  "",
			want:    "",
		},
		{
			name:    "plain key stays key",
			baseURL: "http://localhost:9000/public",
			stored:  "categories/1/icon.png",
			want:    "categories/1/icon.png",
		},
		{
			name:    "extracts key from matching base",
			baseURL: "http://localhost:9000/public",
			stored:  "http://localhost:9000/public/categories/1/icon.png",
			want:    "categories/1/icon.png",
		},
		{
			name:    "trims trailing slash from base",
			baseURL: "http://localhost:9000/public/",
			stored:  "http://localhost:9000/public/categories/1/icon.png",
			want:    "categories/1/icon.png",
		},
		{
			name:    "mismatched base returns empty",
			baseURL: "http://localhost:9000/public",
			stored:  "http://example.com/categories/1/icon.png",
			want:    "",
		},
		{
			name:    "absolute URL without base returns empty",
			baseURL: "",
			stored:  "http://localhost:9000/public/categories/1/icon.png",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := objectKeyFromStoredURL(tt.baseURL, tt.stored)
			if got != tt.want {
				t.Fatalf("objectKeyFromStoredURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestObjectURLRoundTrip(t *testing.T) {
	t.Parallel()

	baseURL := "http://localhost:9000/public/"
	keys := []string{
		"users/user-1-avatar.png",
		"incidents/42/photo.jpg",
		"categories/7/icon.jpeg",
	}

	for _, key := range keys {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			stored := buildObjectURL(baseURL, key)
			got := objectKeyFromStoredURL(baseURL, stored)
			if got != key {
				t.Fatalf("round-trip key = %q, want %q", got, key)
			}
		})
	}
}
