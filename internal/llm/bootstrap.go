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

// Bootstrap reads LLM settings and constructs a Provider using the given
// factories. It returns a nil Provider when LLM is not enabled or the
// endpoint is empty. settingKeys should contain the four setting key
// constants: [enabled, provider, endpoint, model].
func Bootstrap(ctx context.Context, settings SettingsReader, settingKeys [4]string, factories map[string]Factory) BootstrapResult {
	enabledStr, err := settings.GetSetting(ctx, settingKeys[0])
	if err != nil || enabledStr != "true" {
		return BootstrapResult{}
	}

	providerName, _ := settings.GetSetting(ctx, settingKeys[1])
	endpoint, _ := settings.GetSetting(ctx, settingKeys[2])
	modelName, _ := settings.GetSetting(ctx, settingKeys[3])

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
