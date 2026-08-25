package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MCPAuthMode string

const (
	MCPAuthLocal  MCPAuthMode = "local"
	MCPAuthBearer MCPAuthMode = "bearer"
)

const (
	defaultMCPListenAddress      = "127.0.0.1:8080"
	defaultMCPEndpoint           = "/mcp"
	defaultMCPLogLevel           = "info"
	defaultMCPReportTimezone     = "UTC"
	defaultMCPMaxPeriodDays      = 3660
	defaultMCPPageSize           = 50
	defaultMCPMaxPageSize        = 100
	defaultMCPMaxChartPoints     = 400
	defaultMCPMaxFilterValues    = 100
	defaultMCPMaxRequestBodyByte = int64(1 << 20)
	defaultMCPStaleAfter         = 24 * time.Hour
	defaultMCPRequestTimeout     = 30 * time.Second
	maxMCPPostgresRows           = 500
	minimumMCPBearerTokenBytes   = 32
)

type MCPConfig struct {
	ListenAddress       string
	Endpoint            string
	DatabaseURL         string
	LogLevel            string
	AuthMode            MCPAuthMode
	UserIDs             []int64
	BearerToken         string
	AllowedOrigins      []string
	ReportTimezone      string
	MaxPeriodDays       int
	DefaultPageSize     int
	MaxPageSize         int
	MaxChartPoints      int
	MaxFilterValues     int
	MaxRequestBodyBytes int64
	StaleAfter          time.Duration
	RequestTimeout      time.Duration
}

func NewMCPConfigFromEnv() (*MCPConfig, error) {
	databaseURL := strings.TrimSpace(os.Getenv("ZENMCP_DB_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DB_URL"))
	}
	config := &MCPConfig{
		ListenAddress: strings.TrimSpace(
			envOrDefault("ZENMCP_LISTEN_ADDRESS", defaultMCPListenAddress),
		),
		Endpoint:    strings.TrimSpace(envOrDefault("ZENMCP_ENDPOINT", defaultMCPEndpoint)),
		DatabaseURL: databaseURL,
		LogLevel: strings.ToLower(
			strings.TrimSpace(envOrDefault("ZENMCP_LOG_LEVEL", defaultMCPLogLevel)),
		),
		AuthMode: MCPAuthMode(
			strings.ToLower(
				strings.TrimSpace(envOrDefault("ZENMCP_AUTH_MODE", string(MCPAuthLocal))),
			),
		),
		BearerToken: os.Getenv("ZENMCP_BEARER_TOKEN"),
		ReportTimezone: strings.TrimSpace(
			envOrDefault("ZENMCP_REPORT_TIMEZONE", defaultMCPReportTimezone),
		),
	}

	var err error
	if config.UserIDs, err = parseMCPUserIDs(os.Getenv("ZENMCP_USER_IDS")); err != nil {
		return nil, err
	}
	if config.AllowedOrigins, err = parseMCPOrigins(
		os.Getenv("ZENMCP_ALLOWED_ORIGINS"),
	); err != nil {
		return nil, err
	}
	if config.MaxPeriodDays, err = envPositiveInt(
		"ZENMCP_MAX_PERIOD_DAYS",
		defaultMCPMaxPeriodDays,
	); err != nil {
		return nil, err
	}
	if config.DefaultPageSize, err = envPositiveInt(
		"ZENMCP_DEFAULT_PAGE_SIZE",
		defaultMCPPageSize,
	); err != nil {
		return nil, err
	}
	if config.MaxPageSize, err = envPositiveInt(
		"ZENMCP_MAX_PAGE_SIZE",
		defaultMCPMaxPageSize,
	); err != nil {
		return nil, err
	}
	if config.MaxChartPoints, err = envPositiveInt(
		"ZENMCP_MAX_CHART_POINTS",
		defaultMCPMaxChartPoints,
	); err != nil {
		return nil, err
	}
	if config.MaxFilterValues, err = envPositiveInt(
		"ZENMCP_MAX_FILTER_VALUES",
		defaultMCPMaxFilterValues,
	); err != nil {
		return nil, err
	}
	if config.MaxRequestBodyBytes, err = envPositiveInt64(
		"ZENMCP_MAX_REQUEST_BODY_BYTES", defaultMCPMaxRequestBodyByte,
	); err != nil {
		return nil, err
	}
	if config.StaleAfter, err = envPositiveDuration(
		"ZENMCP_STALE_AFTER",
		defaultMCPStaleAfter,
	); err != nil {
		return nil, err
	}
	if config.RequestTimeout, err = envPositiveDuration(
		"ZENMCP_REQUEST_TIMEOUT",
		defaultMCPRequestTimeout,
	); err != nil {
		return nil, err
	}

	if err := ValidateMCPConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func ValidateMCPConfig(config *MCPConfig) error {
	if config == nil {
		return errors.New("MCP configuration is required")
	}
	if strings.TrimSpace(config.DatabaseURL) == "" {
		return errors.New("MCP database URL is required")
	}
	if err := validateMCPAddressAndAuth(config); err != nil {
		return err
	}
	if err := validateMCPEndpoint(config.Endpoint); err != nil {
		return err
	}
	if err := validateMCPUserScope(config); err != nil {
		return err
	}
	if err := validateMCPPresentation(config); err != nil {
		return err
	}
	if err := validateMCPLimits(config); err != nil {
		return err
	}
	return validateMCPOrigins(config.AllowedOrigins)
}

func validateMCPAddressAndAuth(config *MCPConfig) error {
	host, port, err := net.SplitHostPort(config.ListenAddress)
	if err != nil || strings.TrimSpace(host) == "" || !validPort(port) {
		return errors.New("MCP listen address must include a host and valid port")
	}
	switch config.AuthMode {
	case MCPAuthLocal:
		if !isLoopbackHost(host) {
			return errors.New("local MCP authentication may only listen on a loopback address")
		}
	case MCPAuthBearer:
		if len([]byte(config.BearerToken)) < minimumMCPBearerTokenBytes {
			return fmt.Errorf(
				"bearer authentication requires a secret of at least %d bytes",
				minimumMCPBearerTokenBytes,
			)
		}
	default:
		return fmt.Errorf("unsupported MCP authentication mode %q", config.AuthMode)
	}
	return nil
}

func validateMCPUserScope(config *MCPConfig) error {
	for _, userID := range config.UserIDs {
		if userID <= 0 {
			return errors.New("authenticated ZenMoney user IDs must be positive")
		}
	}
	return nil
}

func validateMCPPresentation(config *MCPConfig) error {
	if _, err := time.LoadLocation(config.ReportTimezone); err != nil {
		return fmt.Errorf("invalid MCP report timezone %q", config.ReportTimezone)
	}
	switch config.LogLevel {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("invalid MCP log level %q", config.LogLevel)
	}
}

func validateMCPLimits(config *MCPConfig) error {
	if config.MaxPeriodDays <= 0 || config.DefaultPageSize <= 0 || config.MaxPageSize <= 0 ||
		config.MaxChartPoints <= 0 || config.MaxFilterValues <= 0 ||
		config.MaxRequestBodyBytes <= 0 || config.StaleAfter <= 0 || config.RequestTimeout <= 0 {
		return errors.New("MCP limits must be greater than zero")
	}
	if config.DefaultPageSize > config.MaxPageSize {
		return errors.New("default MCP page size must not exceed maximum page size")
	}
	if config.MaxPageSize > maxMCPPostgresRows {
		return fmt.Errorf(
			"maximum MCP page size must not exceed PostgreSQL hard limit %d",
			maxMCPPostgresRows,
		)
	}
	if config.MaxChartPoints > maxMCPPostgresRows {
		return fmt.Errorf(
			"maximum MCP chart points must not exceed PostgreSQL hard limit %d",
			maxMCPPostgresRows,
		)
	}
	return nil
}

func validateMCPOrigins(origins []string) error {
	for _, origin := range origins {
		if _, err := normalizeMCPOrigin(origin); err != nil {
			return err
		}
	}
	return nil
}

func NewMCPLogger(config *MCPConfig) *slog.Logger {
	level := slog.LevelInfo
	switch config.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func envOrDefault(name, fallback string) string {
	if value, exists := os.LookupEnv(name); exists && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func envPositiveInt(name string, fallback int) (int, error) {
	value, err := envPositiveInt64(name, int64(fallback))
	if err != nil {
		return 0, err
	}
	converted := int(value)
	if int64(converted) != value {
		return 0, fmt.Errorf("%s is too large", name)
	}
	return converted, nil
}

func envPositiveInt64(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func envPositiveDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}

func parseMCPUserIDs(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	seen := make(map[int64]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		userID, err := strconv.ParseInt(part, 10, 64)
		if err != nil || userID <= 0 {
			return nil, fmt.Errorf("invalid authenticated ZenMoney user ID %q", part)
		}
		seen[userID] = struct{}{}
	}
	result := make([]int64, 0, len(seen))
	for userID := range seen {
		result = append(result, userID)
	}
	if len(result) == 0 {
		return nil, errors.New("ZENMCP_USER_IDS must contain at least one positive user ID")
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func parseMCPOrigins(raw string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		origin, err := normalizeMCPOrigin(part)
		if err != nil {
			return nil, err
		}
		seen[origin] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for origin := range seen {
		result = append(result, origin)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeMCPOrigin(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("invalid trusted MCP origin %q", value)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func validateMCPEndpoint(endpoint string) error {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Path != endpoint || parsed.RawQuery != "" || parsed.Fragment != "" ||
		!strings.HasPrefix(endpoint, "/") || endpoint == "/" || path.Clean(endpoint) != endpoint ||
		endpoint == "/healthz" || endpoint == "/readyz" {
		return errors.New(
			"MCP endpoint must be a clean absolute path distinct from health endpoints",
		)
	}
	return nil
}

func validPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}
