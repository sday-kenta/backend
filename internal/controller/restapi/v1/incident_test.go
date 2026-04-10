package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/incidenterr"
)

func TestParseIncidentStatusFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   [][]byte
		want    []string
		wantErr error
	}{
		{
			name:  "empty query stays empty",
			input: nil,
			want:  nil,
		},
		{
			name:  "deduplicates and normalizes",
			input: [][]byte{[]byte("review"), []byte(" published "), []byte("review")},
			want:  []string{entity.IncidentStatusReview, entity.IncidentStatusPublished},
		},
		{
			name:  "ignores blank values",
			input: [][]byte{[]byte(""), []byte("draft")},
			want:  []string{entity.IncidentStatusDraft},
		},
		{
			name:    "rejects invalid status",
			input:   [][]byte{[]byte("unknown")},
			wantErr: incidenterr.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseIncidentStatusFilters(tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
