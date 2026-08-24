package mcpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadinessCheck verifies dependencies required to serve analytics requests.
// It is intentionally separate from liveness, which never performs I/O.
type ReadinessCheck func(*http.Request) error

type HTTPOptions struct {
	IdentityResolver IdentityResolver
	ReadinessCheck   ReadinessCheck

	// ProtectOrigin wraps the MCP endpoint with Origin validation. If nil,
	// net/http's deny-by-default CrossOriginProtection is installed.
	ProtectOrigin func(http.Handler) http.Handler

	JSONResponse                 bool
	MaxRequestBodyBytes          int64
	PropagateRequestCancellation bool
}

type HTTPHandlers struct {
	MCP       http.Handler
	Health    http.Handler
	Readiness http.Handler
}

func NewHTTPHandlers(server *Server, options HTTPOptions) (HTTPHandlers, error) {
	if server == nil || server.core == nil {
		return HTTPHandlers{}, errors.New("nil MCP server")
	}
	if options.IdentityResolver == nil {
		return HTTPHandlers{}, errors.New("identity resolver is required")
	}

	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server.core },
		&mcp.StreamableHTTPOptions{
			Stateless:                    true,
			JSONResponse:                 options.JSONResponse,
			MaxRequestBodyBytes:          options.MaxRequestBodyBytes,
			PropagateRequestCancellation: options.PropagateRequestCancellation,
		},
	)

	var endpoint http.Handler = resolveIdentity(options.IdentityResolver, streamable)
	protectOrigin := options.ProtectOrigin
	if protectOrigin == nil {
		protection := http.NewCrossOriginProtection()
		protectOrigin = protection.Handler
	}
	endpoint = protectOrigin(endpoint)

	return HTTPHandlers{
		MCP: endpoint,
		Health: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeStatus(w, http.StatusOK, "ok")
		}),
		Readiness: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if options.ReadinessCheck != nil {
				if err := options.ReadinessCheck(request); err != nil {
					writeStatus(w, http.StatusServiceUnavailable, "not_ready")
					return
				}
			}
			writeStatus(w, http.StatusOK, "ready")
		}),
	}, nil
}

func resolveIdentity(resolver IdentityResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		principal, err := resolver.Resolve(request.Context(), request)
		if err != nil {
			writeStatus(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if err := validatePrincipal(principal); err != nil {
			writeStatus(w, http.StatusForbidden, "invalid_identity")
			return
		}
		next.ServeHTTP(w, request.WithContext(contextWithPrincipal(request.Context(), principal)))
	})
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": value})
}
