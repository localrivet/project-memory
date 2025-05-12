package providers

import (
	"fmt"
)

// ProviderFactory creates and returns appropriate LLM providers
type ProviderFactory struct {
	// ProviderConfigs stores configuration for each provider
	ProviderConfigs map[string]Config
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(configs map[string]Config) *ProviderFactory {
	return &ProviderFactory{
		ProviderConfigs: configs,
	}
}

// GetProvider returns an initialized provider instance for the specified provider name
func (f *ProviderFactory) GetProvider(providerName string) (LLMProvider, error) {
	config, exists := f.ProviderConfigs[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration for provider '%s' not found", providerName)
	}

	// Return appropriate provider based on name
	switch providerName {
	case ProviderAnthropic:
		return NewAnthropicProvider(config), nil
	case ProviderOpenAI:
		return NewOpenAIProvider(config), nil
	case ProviderGoogle:
		return NewGoogleProvider(config), nil
	case ProviderXAI:
		return NewXAIProvider(config), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// GetAllProviders returns all available providers based on configured providers
func (f *ProviderFactory) GetAllProviders() []LLMProvider {
	var providers []LLMProvider

	// Create all configured providers
	for providerName, config := range f.ProviderConfigs {
		// Skip providers with no API key
		if config.APIKey == "" {
			continue
		}

		provider, err := f.GetProvider(providerName)
		if err == nil {
			providers = append(providers, provider)
		}
		// Silently skip providers that couldn't be created
	}

	return providers
}

// GetProviderChain returns an ordered list of providers to try in sequence
// The ordered list is created based on the given preference order
func (f *ProviderFactory) GetProviderChain(preferenceOrder []string) []LLMProvider {
	var chain []LLMProvider

	// First add providers in the preferred order
	for _, name := range preferenceOrder {
		if config, exists := f.ProviderConfigs[name]; exists && config.APIKey != "" {
			if provider, err := f.GetProvider(name); err == nil {
				chain = append(chain, provider)
			}
		}
	}

	// Then add any remaining providers not in the preference list
	for name, config := range f.ProviderConfigs {
		// Skip if no API key or already in the chain
		if config.APIKey == "" {
			continue
		}

		alreadyInChain := false
		for _, prefName := range preferenceOrder {
			if name == prefName {
				alreadyInChain = true
				break
			}
		}

		if !alreadyInChain {
			if provider, err := f.GetProvider(name); err == nil {
				chain = append(chain, provider)
			}
		}
	}

	return chain
}
