package llm

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// SettingsReader reads key-value settings by name.
type SettingsReader interface {
	GetSetting(ctx context.Context, key string) (string, error)
}

// BootstrapResult holds the result of bootstrapping an LLM provider from settings.
type BootstrapResult struct {
	Provider     Provider
	ProviderName string
	ModelName    string
}

// BootstrapSettings holds the database setting-key names used to configure the LLM provider.
type BootstrapSettings struct {
	Enabled  string
	Provider string
	Endpoint string
	Model    string
}

// valid reports whether all required setting-key names are populated.
func (s BootstrapSettings) valid() bool {
	return s.Enabled != "" && s.Provider != "" && s.Endpoint != "" && s.Model != ""
}

// Bootstrap reads LLM settings and constructs a Provider using the given
// factories. It returns a nil Provider when LLM is not enabled or the
// endpoint is empty.
func Bootstrap(ctx context.Context, settings SettingsReader, keys BootstrapSettings, factories map[string]Factory) BootstrapResult {
	if !keys.valid() {
		slog.WarnContext(ctx, "invalid LLM bootstrap settings, AI enrichment disabled")
		return BootstrapResult{}
	}
	enabledStr, err := settings.GetSetting(ctx, keys.Enabled)
	if err != nil || enabledStr != "true" {
		return BootstrapResult{}
	}

	providerName, _ := settings.GetSetting(ctx, keys.Provider)
	endpoint, _ := settings.GetSetting(ctx, keys.Endpoint)
	modelName, _ := settings.GetSetting(ctx, keys.Model)

	if endpoint == "" {
		return BootstrapResult{ProviderName: providerName, ModelName: modelName}
	}

	p, name, err := NewProvider(providerName, endpoint, modelName, factories)
	if err != nil {
		slog.WarnContext(ctx, "unsupported LLM provider, AI enrichment disabled",
			slog.String(otelkeys.Source, providerName),
		)
		return BootstrapResult{ProviderName: providerName, ModelName: modelName}
	}

	return BootstrapResult{Provider: p, ProviderName: name, ModelName: modelName}
}
