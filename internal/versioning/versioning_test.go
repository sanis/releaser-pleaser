package versioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersionPrefix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    VersionPrefix
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "empty defaults to auto",
			input:   "",
			want:    VersionPrefixAuto,
			wantErr: assert.NoError,
		},
		{
			name:    "auto",
			input:   "auto",
			want:    VersionPrefixAuto,
			wantErr: assert.NoError,
		},
		{
			name:    "v",
			input:   "v",
			want:    VersionPrefixV,
			wantErr: assert.NoError,
		},
		{
			name:    "none",
			input:   "none",
			want:    VersionPrefixNone,
			wantErr: assert.NoError,
		},
		{
			name:    "case and whitespace insensitive",
			input:   "  None  ",
			want:    VersionPrefixNone,
			wantErr: assert.NoError,
		},
		{
			name:    "unknown value",
			input:   "prefix",
			want:    VersionPrefixAuto,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionPrefix(tt.input)
			if !tt.wantErr(t, err) {
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionPrefix_String(t *testing.T) {
	tests := []struct {
		prefix VersionPrefix
		want   string
	}{
		{prefix: VersionPrefixAuto, want: "auto"},
		{prefix: VersionPrefixV, want: "v"},
		{prefix: VersionPrefixNone, want: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.prefix.String())

			parsed, err := ParseVersionPrefix(tt.want)
			require.NoError(t, err)
			assert.Equal(t, tt.prefix, parsed)
		})
	}
}
