package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewMCPConfigFromEnvUsesIndependentDefaults(t *testing.T) {
	resetMCPEnvironment(t)
	t.Setenv("DB_URL", "postgres://localhost/zenmoney")
	t.Setenv("ZENMCP_USER_IDS", "9, 4,9")
	t.Setenv("ZEN_API_TOKEN", "")

	config, err := NewMCPConfigFromEnv()

	require.NoError(t, err)
	require.Equal(t, defaultMCPListenAddress, config.ListenAddress)
	require.Equal(t, defaultMCPEndpoint, config.Endpoint)
	require.Equal(t, MCPAuthLocal, config.AuthMode)
	require.Equal(t, []int64{4, 9}, config.UserIDs)
	require.Equal(t, defaultMCPReportTimezone, config.ReportTimezone)
	require.Equal(t, defaultMCPMaxRequestBodyByte, config.MaxRequestBodyBytes)
	require.Equal(t, defaultMCPStaleAfter, config.StaleAfter)
	require.Equal(t, defaultMCPRequestTimeout, config.RequestTimeout)
}

func TestMCPPrefixedDatabaseURLTakesPriority(t *testing.T) {
	resetMCPEnvironment(t)
	t.Setenv("ZENMCP_DB_URL", "postgres://mcp/zenmoney")
	t.Setenv("DB_URL", "postgres://sync/zenmoney")
	t.Setenv("ZENMCP_USER_IDS", "1")

	config, err := NewMCPConfigFromEnv()

	require.NoError(t, err)
	require.Equal(t, "postgres://mcp/zenmoney", config.DatabaseURL)
}

func TestNewMCPConfigFromEnvReadsLimitsOriginsAndBearerMode(t *testing.T) {
	resetMCPEnvironment(t)
	t.Setenv("ZENMCP_DB_URL", "postgres://localhost/zenmoney")
	t.Setenv("ZENMCP_LISTEN_ADDRESS", "0.0.0.0:9090")
	t.Setenv("ZENMCP_ENDPOINT", "/api/mcp")
	t.Setenv("ZENMCP_AUTH_MODE", "bearer")
	t.Setenv("ZENMCP_BEARER_TOKEN", "0123456789abcdef0123456789abcdef")
	t.Setenv("ZENMCP_USER_IDS", "42")
	t.Setenv("ZENMCP_ALLOWED_ORIGINS", "https://app.example/, http://localhost:3000,https://app.example")
	t.Setenv("ZENMCP_REPORT_TIMEZONE", "Europe/Moscow")
	t.Setenv("ZENMCP_LOG_LEVEL", "debug")
	t.Setenv("ZENMCP_MAX_PERIOD_DAYS", "90")
	t.Setenv("ZENMCP_DEFAULT_PAGE_SIZE", "20")
	t.Setenv("ZENMCP_MAX_PAGE_SIZE", "40")
	t.Setenv("ZENMCP_MAX_CHART_POINTS", "80")
	t.Setenv("ZENMCP_MAX_FILTER_VALUES", "30")
	t.Setenv("ZENMCP_MAX_REQUEST_BODY_BYTES", "2048")
	t.Setenv("ZENMCP_STALE_AFTER", "6h")
	t.Setenv("ZENMCP_REQUEST_TIMEOUT", "45s")

	config, err := NewMCPConfigFromEnv()

	require.NoError(t, err)
	require.Equal(t, MCPAuthBearer, config.AuthMode)
	require.Equal(t, []string{"http://localhost:3000", "https://app.example"}, config.AllowedOrigins)
	require.Equal(t, "Europe/Moscow", config.ReportTimezone)
	require.Equal(t, 90, config.MaxPeriodDays)
	require.Equal(t, 20, config.DefaultPageSize)
	require.Equal(t, 40, config.MaxPageSize)
	require.Equal(t, 80, config.MaxChartPoints)
	require.Equal(t, 30, config.MaxFilterValues)
	require.Equal(t, int64(2048), config.MaxRequestBodyBytes)
	require.Equal(t, 6*time.Hour, config.StaleAfter)
	require.Equal(t, 45*time.Second, config.RequestTimeout)
}

func TestValidateMCPConfigRejectsUnsafeAuthenticationConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*MCPConfig)
		want   string
	}{
		{
			name:   "local on wildcard",
			mutate: func(config *MCPConfig) { config.ListenAddress = "0.0.0.0:8080" },
			want:   "loopback",
		},
		{
			name:   "local on external address",
			mutate: func(config *MCPConfig) { config.ListenAddress = "192.0.2.1:8080" },
			want:   "loopback",
		},
		{
			name: "bearer without secret",
			mutate: func(config *MCPConfig) {
				config.AuthMode = MCPAuthBearer
				config.ListenAddress = "0.0.0.0:8080"
			},
			want: "at least 32 bytes",
		},
		{
			name:   "missing users",
			mutate: func(config *MCPConfig) { config.UserIDs = nil },
			want:   "at least one",
		},
		{
			name:   "unknown mode",
			mutate: func(config *MCPConfig) { config.AuthMode = "oauth-ish" },
			want:   "unsupported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validMCPConfig()
			test.mutate(config)
			require.ErrorContains(t, ValidateMCPConfig(config), test.want)
		})
	}
}

func TestValidateMCPConfigRejectsInvalidPathsOriginsAndLimits(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*MCPConfig)
		want   string
	}{
		{name: "missing database", mutate: func(config *MCPConfig) { config.DatabaseURL = "" }, want: "database URL"},
		{name: "relative endpoint", mutate: func(config *MCPConfig) { config.Endpoint = "mcp" }, want: "absolute path"},
		{name: "health collision", mutate: func(config *MCPConfig) { config.Endpoint = "/healthz" }, want: "health endpoints"},
		{name: "query endpoint", mutate: func(config *MCPConfig) { config.Endpoint = "/mcp?x=1" }, want: "absolute path"},
		{name: "origin with path", mutate: func(config *MCPConfig) { config.AllowedOrigins = []string{"https://example.com/path"} }, want: "invalid trusted"},
		{name: "javascript origin", mutate: func(config *MCPConfig) { config.AllowedOrigins = []string{"javascript:alert(1)"} }, want: "invalid trusted"},
		{name: "invalid timezone", mutate: func(config *MCPConfig) { config.ReportTimezone = "Mars/Base" }, want: "timezone"},
		{name: "invalid log level", mutate: func(config *MCPConfig) { config.LogLevel = "trace" }, want: "log level"},
		{name: "page sizes reversed", mutate: func(config *MCPConfig) { config.DefaultPageSize = 101 }, want: "must not exceed"},
		{name: "page limit above store hard limit", mutate: func(config *MCPConfig) { config.MaxPageSize = 501 }, want: "PostgreSQL hard limit 500"},
		{name: "chart limit above store hard limit", mutate: func(config *MCPConfig) { config.MaxChartPoints = 501 }, want: "PostgreSQL hard limit 500"},
		{name: "zero body limit", mutate: func(config *MCPConfig) { config.MaxRequestBodyBytes = 0 }, want: "greater than zero"},
		{name: "zero request timeout", mutate: func(config *MCPConfig) { config.RequestTimeout = 0 }, want: "greater than zero"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validMCPConfig()
			test.mutate(config)
			require.ErrorContains(t, ValidateMCPConfig(config), test.want)
		})
	}
}

func TestNewMCPConfigFromEnvRejectsMalformedValuesWithoutLeakingSecret(t *testing.T) {
	resetMCPEnvironment(t)
	t.Setenv("ZENMCP_DB_URL", "postgres://localhost/zenmoney")
	t.Setenv("ZENMCP_USER_IDS", "not-an-id")
	t.Setenv("ZENMCP_BEARER_TOKEN", "do-not-leak")
	t.Setenv("ZENMCP_MAX_PAGE_SIZE", "unlimited")

	_, err := NewMCPConfigFromEnv()

	require.Error(t, err)
	require.NotContains(t, err.Error(), "do-not-leak")
}

func TestNewMCPConfigFromEnvRejectsShortBearerWithoutLeakingSecret(t *testing.T) {
	resetMCPEnvironment(t)
	t.Setenv("ZENMCP_DB_URL", "postgres://localhost/zenmoney")
	t.Setenv("ZENMCP_LISTEN_ADDRESS", "0.0.0.0:8080")
	t.Setenv("ZENMCP_AUTH_MODE", "bearer")
	t.Setenv("ZENMCP_USER_IDS", "1")
	secret := "short-private-secret"
	t.Setenv("ZENMCP_BEARER_TOKEN", secret)

	_, err := NewMCPConfigFromEnv()

	require.ErrorContains(t, err, "at least 32 bytes")
	require.NotContains(t, err.Error(), secret)
}

func TestNewMCPConfigFromEnvRejectsNonPositiveRequestTimeout(t *testing.T) {
	resetMCPEnvironment(t)
	t.Setenv("ZENMCP_DB_URL", "postgres://localhost/zenmoney")
	t.Setenv("ZENMCP_USER_IDS", "1")
	t.Setenv("ZENMCP_REQUEST_TIMEOUT", "0s")

	_, err := NewMCPConfigFromEnv()

	require.ErrorContains(t, err, "ZENMCP_REQUEST_TIMEOUT must be a positive duration")
}

func validMCPConfig() *MCPConfig {
	return &MCPConfig{
		ListenAddress: "127.0.0.1:8080", Endpoint: "/mcp",
		DatabaseURL: "postgres://localhost/zenmoney", LogLevel: "info",
		AuthMode: MCPAuthLocal, UserIDs: []int64{1}, ReportTimezone: "UTC",
		MaxPeriodDays: 365, DefaultPageSize: 25, MaxPageSize: 100,
		MaxChartPoints: 200, MaxFilterValues: 50, MaxRequestBodyBytes: 1024,
		StaleAfter: time.Hour, RequestTimeout: 30 * time.Second,
	}
}

func resetMCPEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ZENMCP_DB_URL", "DB_URL", "ZENMCP_LISTEN_ADDRESS", "ZENMCP_ENDPOINT",
		"ZENMCP_LOG_LEVEL", "ZENMCP_AUTH_MODE", "ZENMCP_USER_IDS", "ZENMCP_BEARER_TOKEN",
		"ZENMCP_ALLOWED_ORIGINS", "ZENMCP_REPORT_TIMEZONE", "ZENMCP_MAX_PERIOD_DAYS",
		"ZENMCP_DEFAULT_PAGE_SIZE", "ZENMCP_MAX_PAGE_SIZE", "ZENMCP_MAX_CHART_POINTS",
		"ZENMCP_MAX_FILTER_VALUES", "ZENMCP_MAX_REQUEST_BODY_BYTES", "ZENMCP_STALE_AFTER",
		"ZENMCP_REQUEST_TIMEOUT",
	} {
		t.Setenv(name, "")
	}
}
