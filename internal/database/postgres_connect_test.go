package database

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConnect_UDSDSNParsing(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		wantHost string
		wantPort uint16
	}{
		{
			name:     "Unix socket key-value DSN",
			dsn:      "host=/var/run/postgresql port=5430 dbname=ad_event_processor user=postgres",
			wantHost: "/var/run/postgresql",
			wantPort: 5430,
		},
		{
			name:     "Unix socket URL DSN",
			dsn:      "postgres://postgres:pass@/ad_event_processor?host=/var/run/postgresql&port=5430",
			wantHost: "/var/run/postgresql",
			wantPort: 5430,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := pgxpool.ParseConfig(tt.dsn)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, config.ConnConfig.Host)
			assert.Equal(t, tt.wantPort, config.ConnConfig.Port)
		})
	}
}
