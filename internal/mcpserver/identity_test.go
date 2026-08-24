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

const validBearerToken = "0123456789abcdef0123456789abcdef"

func TestBearerIdentityResolver(t *testing.T) {
	principal := analytics.Principal{Subject: "remote", AllUsers: true}
	resolver, err := NewBearerIdentityResolver(validBearerToken, principal)
	require.NoError(t, err)

	t.Run("accepts exact bearer authorization", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		request.Header.Set("Authorization", "Bearer "+validBearerToken)
		actual, err := resolver.Resolve(context.Background(), request)
		require.NoError(t, err)
		assert.Equal(t, principal, actual)
	})

	for name, values := range map[string][]string{
		"missing":          nil,
		"wrong token":      {"Bearer different"},
		"wrong scheme":     {"bearer " + validBearerToken},
		"trailing space":   {"Bearer " + validBearerToken + " "},
		"multiple headers": {"Bearer " + validBearerToken, "Bearer " + validBearerToken},
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
	for _, token := range []string{
		"",
		"0123456789abcdef0123456789abcde",
		" 0123456789abcdef0123456789abcdef",
		"0123456789abcdef0123456789abcdef ",
		"0123456789abcdef 123456789abcdef",
		"0123456789abcdef0123456789abcde\n",
	} {
		resolver, err := NewBearerIdentityResolver(token, principal)
		assert.Error(t, err)
		assert.Nil(t, resolver)
	}

	resolver, err := NewBearerIdentityResolver(validBearerToken, analytics.Principal{})
	assert.ErrorIs(t, err, ErrInvalidPrincipal)
	assert.Nil(t, resolver)
}

func TestValidatePrincipalRequiresExactlyOneUserScope(t *testing.T) {
	for name, principal := range map[string]analytics.Principal{
		"empty scope":   {Subject: "subject"},
		"both scopes":   {Subject: "subject", AllUsers: true, UserIDs: []int64{1}},
		"empty subject": {AllUsers: true},
		"invalid ID":    {Subject: "subject", UserIDs: []int64{0}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, validatePrincipal(principal), ErrInvalidPrincipal)
		})
	}

	assert.NoError(t, validatePrincipal(analytics.Principal{Subject: "all", AllUsers: true}))
	assert.NoError(t, validatePrincipal(analytics.Principal{
		Subject: "restricted", UserIDs: []int64{1, 2},
	}))
}
