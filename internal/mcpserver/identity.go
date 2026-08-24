package mcpserver

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
)

var (
	ErrUnauthenticated  = errors.New("request is not authenticated")
	ErrInvalidPrincipal = errors.New("resolved identity has no subject or users")
)

// IdentityResolver is the authentication boundary for the MCP HTTP endpoint.
// Implementations must validate credentials and derive the subject and allowed
// ZenMoney users. Tool arguments never contain a trusted user identifier.
type IdentityResolver interface {
	Resolve(context.Context, *http.Request) (analytics.Principal, error)
}

type IdentityResolverFunc func(context.Context, *http.Request) (analytics.Principal, error)

func (f IdentityResolverFunc) Resolve(ctx context.Context, request *http.Request) (analytics.Principal, error) {
	return f(ctx, request)
}

// StaticIdentityResolver is intended for explicit local-development wiring and
// tests. Production code should use a resolver backed by verified credentials.
type StaticIdentityResolver struct {
	Principal analytics.Principal
}

func (r StaticIdentityResolver) Resolve(context.Context, *http.Request) (analytics.Principal, error) {
	return r.Principal, nil
}

// BearerIdentityResolver authenticates a remote single-principal deployment.
// It retains only a digest of the configured Authorization value and compares
// request digests in constant time. Callers must never put the token in logs or
// error messages.
type BearerIdentityResolver struct {
	authorizationDigest [sha256.Size]byte
	principal           analytics.Principal
}

func NewBearerIdentityResolver(token string, principal analytics.Principal) (*BearerIdentityResolver, error) {
	if token == "" || strings.TrimSpace(token) != token || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("bearer token must be non-empty and contain no whitespace")
	}
	if err := validatePrincipal(principal); err != nil {
		return nil, err
	}
	return &BearerIdentityResolver{
		authorizationDigest: sha256.Sum256([]byte("Bearer " + token)),
		principal:           principal,
	}, nil
}

func (r *BearerIdentityResolver) Resolve(_ context.Context, request *http.Request) (analytics.Principal, error) {
	if r == nil || request == nil {
		return analytics.Principal{}, ErrUnauthenticated
	}
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		return analytics.Principal{}, ErrUnauthenticated
	}
	actualDigest := sha256.Sum256([]byte(values[0]))
	if subtle.ConstantTimeCompare(actualDigest[:], r.authorizationDigest[:]) != 1 {
		return analytics.Principal{}, ErrUnauthenticated
	}
	return r.principal, nil
}

type principalContextKey struct{}

func contextWithPrincipal(ctx context.Context, principal analytics.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func principalFromContext(ctx context.Context) (analytics.Principal, error) {
	principal, ok := ctx.Value(principalContextKey{}).(analytics.Principal)
	if !ok || principal.Subject == "" || len(principal.UserIDs) == 0 {
		return analytics.Principal{}, ErrUnauthenticated
	}
	return principal, nil
}

func validatePrincipal(principal analytics.Principal) error {
	if principal.Subject == "" || len(principal.UserIDs) == 0 {
		return ErrInvalidPrincipal
	}
	return nil
}
