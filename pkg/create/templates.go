package create

import (
	"fmt"
	"slices"
	"sort"
)

// Template key constants
const (
	TemplateSampleApp      = "sample-app"
	TemplateAdvancedSample = "advanced-sample"
	TemplateComputerUse    = "computer-use"
	TemplateCUA            = "cua"
	TemplateMagnitude      = "magnitude"
	TemplateGeminiCUA      = "gemini-cua"
	TemplateBrowserUse     = "browser-use"
	TemplateStagehand      = "stagehand"
	TemplateOAGICUA        = "oagi-cua"
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
		Description: "Implements basic Kernel apps",
		Languages:   []string{LanguageTypeScript, LanguagePython},
	},
	TemplateAdvancedSample: {
		Name:        "Advanced Sample",
		Description: "Implements sample actions with advanced Kernel configs",
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
	TemplateOAGICUA: {
		Name:        "OAGI CUA",
		Description: "Implements OpenAGI's Lux computer-use models",
		Languages:   []string{LanguagePython},
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
	EnvVars       []string
	InvokeCommand string
}

var Commands = map[string]map[string]DeployConfig{
	LanguageTypeScript: {
		TemplateSampleApp: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{},
			InvokeCommand: `kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'`,
		},
		TemplateAdvancedSample: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{},
			InvokeCommand: "kernel invoke ts-advanced test-captcha-solver",
		},
		TemplateStagehand: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{"OPENAI_API_KEY=XXX"},
			InvokeCommand: `kernel invoke ts-stagehand teamsize-task --payload '{"company": "Kernel"}'`,
		},
		TemplateComputerUse: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{"ANTHROPIC_API_KEY=XXX"},
			InvokeCommand: `kernel invoke ts-cu cu-task --payload '{"query": "Return the first url of a search result for NYC restaurant reviews Pete Wells"}'`,
		},
		TemplateMagnitude: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{"ANTHROPIC_API_KEY=XXX"},
			InvokeCommand: `kernel invoke ts-magnitude mag-url-extract --payload '{"url": "https://en.wikipedia.org/wiki/Special:Random"}'`,
		},
		TemplateCUA: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{"OPENAI_API_KEY=XXX"},
			InvokeCommand: `kernel invoke ts-cua cua-task --payload '{"task": "Go to https://news.ycombinator.com and get the top 5 articles"}'`,
		},
		TemplateGeminiCUA: {
			EntryPoint:    "index.ts",
			EnvVars:       []string{"GOOGLE_API_KEY=XXX", "OPENAI_API_KEY=XXX"},
			InvokeCommand: "kernel invoke ts-gemini-cua gemini-cua-task",
		},
	},
	LanguagePython: {
		TemplateSampleApp: {
			EntryPoint:    "main.py",
			EnvVars:       []string{},
			InvokeCommand: `kernel invoke python-basic get-page-title --payload '{"url": "https://www.google.com"}'`,
		},
		TemplateAdvancedSample: {
			EntryPoint:    "main.py",
			EnvVars:       []string{},
			InvokeCommand: "kernel invoke python-advanced test-captcha-solver",
		},
		TemplateBrowserUse: {
			EntryPoint:    "main.py",
			EnvVars:       []string{"OPENAI_API_KEY=XXX"},
			InvokeCommand: `kernel invoke python-bu bu-task --payload '{"task": "Compare the price of gpt-4o and DeepSeek-V3"}'`,
		},
		TemplateComputerUse: {
			EntryPoint:    "main.py",
			EnvVars:       []string{"ANTHROPIC_API_KEY=XXX"},
			InvokeCommand: `kernel invoke python-cu cu-task --payload '{"query": "Return the first url of a search result for NYC restaurant reviews Pete Wells"}'`,
		},
		TemplateCUA: {
			EntryPoint:    "main.py",
			EnvVars:       []string{"OPENAI_API_KEY=XXX"},
			InvokeCommand: `kernel invoke python-cua cua-task --payload '{"task": "Go to https://news.ycombinator.com and get the top 5 articles"}'`,
		},
		TemplateOAGICUA: {
			EntryPoint:    "main.py",
			EnvVars:       []string{"OAGI_API_KEY=XXX"},
			InvokeCommand: `kernel invoke python-oagi-cua oagi-default-task --payload '{"instruction": "Navigate to https://agiopen.org"}'`,
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
	for _, env := range config.EnvVars {
		cmd += " --env " + env
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
