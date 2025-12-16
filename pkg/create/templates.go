package create

import (
	"fmt"
	"slices"
	"sort"
)

// Template key constants
const (
	TemplateSampleApp     = "sample-app"
	TemplateCaptchaSolver = "captcha-solver"
	TemplateComputerUse   = "computer-use"
	TemplateCUA           = "cua"
	TemplateMagnitude     = "magnitude"
	TemplateGeminiCUA     = "gemini-cua"
	TemplateBrowserUse    = "browser-use"
	TemplateStagehand     = "stagehand"
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
	TemplateSampleApp: {
		Name:        "Sample App",
		Description: "Implements a basic Kernel app",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	TemplateCaptchaSolver: {
		Name:        "CAPTCHA Solver",
		Description: "Demo of Kernel's auto-CAPTCHA solving capability",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	TemplateComputerUse: {
		Name:        "Computer Use",
		Description: "Implements the Anthropic Computer Use SDK",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	TemplateCUA: {
		Name:        "CUA Sample",
		Description: "Implements a Computer Use Agent (OpenAI CUA) sample",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	TemplateMagnitude: {
		Name:        "Magnitude",
		Description: "Implements the Magnitude.run SDK",
		Languages:   []string{LanguageTypeScript},
	},
	TemplateGeminiCUA: {
		Name:        "Gemini CUA",
		Description: "Implements Gemini 2.5 Computer Use Agent",
		Languages:   []string{LanguageTypeScript},
	},
	TemplateBrowserUse: {
		Name:        "Browser Use",
		Description: "Implements Browser Use SDK",
		Languages:   []string{LanguagePython},
	},
	TemplateStagehand: {
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

	sort.Slice(templates, func(i, j int) bool {
		// Put computer-use first, then sort alphabetically
		if templates[i].Key == TemplateComputerUse {
			return true
		}
		if templates[j].Key == TemplateComputerUse {
			return false
		}
		return templates[i].Key < templates[j].Key
	})

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

type DeployConfig struct {
	EntryPoint    string
	NeedsEnvFile  bool
	InvokeCommand string
}

var Commands = map[string]map[string]DeployConfig{
	LanguageTypeScript: {
		TemplateSampleApp: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  false,
			InvokeCommand: `kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'`,
		},
		TemplateCaptchaSolver: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  false,
			InvokeCommand: "kernel invoke ts-captcha-solver test-captcha-solver",
		},
		TemplateStagehand: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke ts-stagehand teamsize-task --payload '{"company": "Kernel"}'`,
		},
		TemplateComputerUse: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke ts-cu cu-task --payload '{"query": "Return the first url of a search result for NYC restaurant reviews Pete Wells"}'`,
		},
		TemplateMagnitude: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke ts-magnitude mag-url-extract --payload '{"url": "https://en.wikipedia.org/wiki/Special:Random"}'`,
		},
		TemplateCUA: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke ts-cua cua-task --payload '{"task": "Go to https://news.ycombinator.com and get the top 5 articles"}'`,
		},
		TemplateGeminiCUA: {
			EntryPoint:    "index.ts",
			NeedsEnvFile:  true,
			InvokeCommand: "kernel invoke ts-gemini-cua gemini-cua-task",
		},
	},
	LanguagePython: {
		TemplateSampleApp: {
			EntryPoint:    "main.py",
			NeedsEnvFile:  false,
			InvokeCommand: `kernel invoke python-basic get-page-title --payload '{"url": "https://www.google.com"}'`,
		},
		TemplateCaptchaSolver: {
			EntryPoint:    "main.py",
			NeedsEnvFile:  false,
			InvokeCommand: "kernel invoke python-captcha-solver test-captcha-solver",
		},
		TemplateBrowserUse: {
			EntryPoint:    "main.py",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke python-bu bu-task --payload '{"task": "Compare the price of gpt-4o and DeepSeek-V3"}'`,
		},
		TemplateComputerUse: {
			EntryPoint:    "main.py",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke python-cu cu-task --payload '{"query": "Return the first url of a search result for NYC restaurant reviews Pete Wells"}'`,
		},
		TemplateCUA: {
			EntryPoint:    "main.py",
			NeedsEnvFile:  true,
			InvokeCommand: `kernel invoke python-cua cua-task --payload '{"task": "Go to https://news.ycombinator.com and get the top 5 articles"}'`,
		},
	},
}

// GetDeployCommand returns the full deploy command string for a given language and template
func GetDeployCommand(language, template string) string {
	langCommands, ok := Commands[language]
	if !ok {
		return ""
	}

	config, ok := langCommands[template]
	if !ok {
		return ""
	}

	cmd := "kernel deploy " + config.EntryPoint
	if config.NeedsEnvFile {
		cmd += " --env-file .env"
	}

	return cmd
}

// GetInvokeSample returns the sample invoke command for a given language and template
func GetInvokeSample(language, template string) string {
	langSamples, ok := Commands[language]
	if !ok {
		return ""
	}

	config, ok := langSamples[template]
	if !ok {
		return ""
	}

	return config.InvokeCommand
}
