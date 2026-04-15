package llm

import "fmt"

// Factory creates a Provider from an endpoint and model name.
type Factory func(endpoint, model string) Provider

// NewProvider instantiates a Provider using the given factory registry.
// If providerName is empty, it defaults to ProviderOllama.
// Returns the instantiated Provider and the resolved provider name.
func NewProvider(providerName, endpoint, model string, factories map[string]Factory) (Provider, string, error) {
	if providerName == "" {
		providerName = ProviderOllama
	}
	factory, ok := factories[providerName]
	if !ok {
		return nil, providerName, fmt.Errorf("unsupported LLM provider: %s", providerName)
	}
	return factory(endpoint, model), providerName, nil
}
