package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
	"github.com/stretchr/testify/require"
)

func TestBuildIdentityResolverDefaultsToAllDatabaseUsers(t *testing.T) {
	resolver, err := buildIdentityResolver(&config.MCPConfig{AuthMode: config.MCPAuthLocal})
	require.NoError(t, err)

	principal, err := resolver.Resolve(
		context.Background(),
		httptest.NewRequest(http.MethodPost, "/mcp", nil),
	)
	require.NoError(t, err)
	require.Equal(t, analytics.Principal{
		Subject: "local-development", AllUsers: true,
	}, principal)
}

func TestBuildIdentityResolverPreservesRestrictiveUserAllowlist(t *testing.T) {
	resolver, err := buildIdentityResolver(&config.MCPConfig{
		AuthMode: config.MCPAuthLocal,
		UserIDs:  []int64{4, 9},
	})
	require.NoError(t, err)

	principal, err := resolver.Resolve(
		context.Background(),
		httptest.NewRequest(http.MethodPost, "/mcp", nil),
	)
	require.NoError(t, err)
	require.Equal(t, analytics.Principal{
		Subject: "local-development", UserIDs: []int64{4, 9},
	}, principal)
}

func TestBuildBearerIdentityResolverDefaultsToAllDatabaseUsers(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	resolver, err := buildIdentityResolver(&config.MCPConfig{
		AuthMode: config.MCPAuthBearer, BearerToken: token,
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	principal, err := resolver.Resolve(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, analytics.Principal{
		Subject: "configured-bearer", AllUsers: true,
	}, principal)
}
