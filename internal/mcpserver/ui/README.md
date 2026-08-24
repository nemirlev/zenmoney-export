# Finance chart MCP App wire contract

`chart.html` is a self-contained MCP App view. It deliberately has no runtime
dependency on a host-specific global object or on external scripts, styles,
fonts, images, or network endpoints.

The view speaks JSON-RPC 2.0 directly over `window.parent.postMessage`, following
the stable MCP Apps `2026-01-26` contract:

1. View sends `ui/initialize` with `appInfo`, `appCapabilities`, and
   `protocolVersion`.
2. Host returns its capabilities and context; view sends
   `ui/notifications/initialized`.
3. Host sends `ui/notifications/tool-input` (and optionally partial input), then
   `ui/notifications/tool-result`. The result's `structuredContent` drives the
   chart; its standard text `content` remains the non-App fallback.
4. View may proxy an approved same-server tool through `tools/call`.
5. View reports size changes and acknowledges `ui/resource-teardown`.

The listener accepts JSON-RPC messages only from `window.parent`. The resource
declares empty CSP origin lists, and the view itself performs no direct network
I/O. Hosts remain responsible for sandboxing the view and authorizing any
view-originated tool call.

Primary references:

- <https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx>
- <https://github.com/modelcontextprotocol/ext-apps/blob/main/src/spec.types.ts>
