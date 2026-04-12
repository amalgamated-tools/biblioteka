---
observability:
  otlp:
    # The opentelemetry section is only included in the MCP gateway config when
    # both GH_AW_OTEL_ENDPOINT and GH_AW_OTEL_HEADERS are non-empty. MCP Gateway
    # v0.2.17+ rejects empty endpoint/headers values, so the section must be
    # omitted unless both secrets are configured together.
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    headers: ${{ secrets.GH_AW_OTEL_HEADERS }}
---
