package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBearerIdentityResolver(t *testing.T) {
	principal := analytics.Principal{Subject: "remote", UserIDs: []int64{42}}
	resolver, err := NewBearerIdentityResolver("secret-token", principal)
	require.NoError(t, err)

	t.Run("accepts exact bearer authorization", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		request.Header.Set("Authorization", "Bearer secret-token")
		actual, err := resolver.Resolve(context.Background(), request)
		require.NoError(t, err)
		assert.Equal(t, principal, actual)
	})

	for name, values := range map[string][]string{
		"missing":          nil,
		"wrong token":      {"Bearer different"},
		"wrong scheme":     {"bearer secret-token"},
		"trailing space":   {"Bearer secret-token "},
		"multiple headers": {"Bearer secret-token", "Bearer secret-token"},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			for _, value := range values {
				request.Header.Add("Authorization", value)
			}
			_, err := resolver.Resolve(context.Background(), request)
			assert.ErrorIs(t, err, ErrUnauthenticated)
		})
	}
}

func TestNewBearerIdentityResolverRejectsUnsafeConfiguration(t *testing.T) {
	principal := analytics.Principal{Subject: "remote", UserIDs: []int64{42}}
	for _, token := range []string{"", " token", "token ", "two tokens", "token\nvalue"} {
		resolver, err := NewBearerIdentityResolver(token, principal)
		assert.Error(t, err)
		assert.Nil(t, resolver)
	}

	resolver, err := NewBearerIdentityResolver("secret-token", analytics.Principal{})
	assert.ErrorIs(t, err, ErrInvalidPrincipal)
	assert.Nil(t, resolver)
}
