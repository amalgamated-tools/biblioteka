---
observability:
  otlp:
    # The opentelemetry section is only included in the MCP gateway config when
    # GH_AW_OTEL_ENDPOINT is non-empty. MCP Gateway v0.2.17+ rejects empty
    # endpoint/headers values, so the section must be omitted when OTEL is not configured.
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    headers: ${{ secrets.GH_AW_OTEL_HEADERS }}
---
