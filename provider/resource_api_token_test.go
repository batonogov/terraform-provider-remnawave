package provider

import (
	"testing"
)

func TestDeriveExpiresInDays(t *testing.T) {
	tests := []struct {
		name      string
		createdAt string
		expireAt  string
		want      int64
	}{
		{
			name:      "30 days",
			createdAt: "2025-01-01T00:00:00.000Z",
			expireAt:  "2025-01-31T00:00:00.000Z",
			want:      30,
		},
		{
			name:      "1 day",
			createdAt: "2025-06-01T12:00:00.000Z",
			expireAt:  "2025-06-02T12:00:00.000Z",
			want:      1,
		},
		{
			name:      "7 days",
			createdAt: "2025-03-10T08:00:00.000Z",
			expireAt:  "2025-03-17T08:00:00.000Z",
			want:      7,
		},
		{
			name:      "without milliseconds",
			createdAt: "2025-01-01T00:00:00Z",
			expireAt:  "2025-02-01T00:00:00Z",
			want:      31,
		},
		{
			name:      "empty createdAt",
			createdAt: "",
			expireAt:  "2025-01-31T00:00:00.000Z",
			want:      0,
		},
		{
			name:      "empty expireAt",
			createdAt: "2025-01-01T00:00:00.000Z",
			expireAt:  "",
			want:      0,
		},
		{
			name:      "invalid format",
			createdAt: "not-a-date",
			expireAt:  "2025-01-31T00:00:00.000Z",
			want:      0,
		},
		{
			name:      "negative duration (expire before create)",
			createdAt: "2025-02-01T00:00:00.000Z",
			expireAt:  "2025-01-01T00:00:00.000Z",
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveExpiresInDays(tt.createdAt, tt.expireAt)
			if got != tt.want {
				t.Errorf("deriveExpiresInDays(%q, %q) = %d, want %d", tt.createdAt, tt.expireAt, got, tt.want)
			}
		})
	}
}
