package create

import (
	"fmt"
	"slices"
)

type TemplateInfo struct {
	Name        string
	Description string
	Languages   []string
}

type TemplateKeyValue struct {
	Key   string
	Value string
}

type TemplateKeyValues []TemplateKeyValue

var Templates = map[string]TemplateInfo{
	"sample-app": {
		Name:        "Sample App",
		Description: "Implements basic Kernel apps",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	"advanced-sample": {
		Name:        "Advanced Sample",
		Description: "Implements sample actions with advanced Kernel configs",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	"computer-use": {
		Name:        "Computer Use",
		Description: "Implements the Anthropic Computer Use SDK",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	"cua": {
		Name:        "CUA Sample",
		Description: "Implements a Computer Use Agent (OpenAI CUA) sample",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	"magnitude": {
		Name:        "Magnitude",
		Description: "Implements the Magnitude.run SDK",
		Languages:   []string{LanguageTypeScript},
	},
	"gemini-cua": {
		Name:        "Gemini CUA",
		Description: "Implements Gemini 2.5 Computer Use Agent",
		Languages:   []string{LanguageTypeScript},
	},
	"browser-use": {
		Name:        "Browser Use",
		Description: "Implements Browser Use SDK",
		Languages:   []string{LanguagePython},
	},
	"stagehand": {
		Name:        "Stagehand",
		Description: "Implements the Stagehand v3 SDK",
		Languages:   []string{LanguageTypeScript},
	},
}

// GetSupportedTemplatesForLanguage returns a list of all supported template names for a given language
func GetSupportedTemplatesForLanguage(language string) TemplateKeyValues {
	templates := make(TemplateKeyValues, 0, len(Templates))
	for tn := range Templates {
		if slices.Contains(Templates[tn].Languages, language) {
			templates = append(templates, TemplateKeyValue{
				Key:   tn,
				Value: fmt.Sprintf("%s - %s", Templates[tn].Name, Templates[tn].Description),
			})
		}
	}
	return templates
}

// GetTemplateDisplayValues extracts display values from TemplateKeyValue slice
func (tkv TemplateKeyValues) GetTemplateDisplayValues() []string {
	options := make([]string, len(tkv))
	for i, kv := range tkv {
		options[i] = kv.Value
	}
	return options
}

// GetTemplateKeyFromValue maps the selected display value back to the template key
func (tkv TemplateKeyValues) GetTemplateKeyFromValue(selectedValue string) (string, error) {
	for _, kv := range tkv {
		if kv.Value == selectedValue {
			return kv.Key, nil
		}
	}
	return "", fmt.Errorf("template not found: %s", selectedValue)
}

// ContainsKey checks if a template key exists in the TemplateKeyValues
func (tkv TemplateKeyValues) ContainsKey(key string) bool {
	for _, kv := range tkv {
		if kv.Key == key {
			return true
		}
	}
	return false
}
