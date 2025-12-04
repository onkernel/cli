package create

const (
	DefaultAppName = "my-kernel-app"
	AppNamePrompt  = "What is the name of your project?"
	LanguagePrompt = "Choose a programming language:"
	TemplatePrompt = "Select a template:"
)

type Language string

const (
	LanguageTypeScript          = "typescript"
	LanguagePython              = "python"
	LanguageShorthandTypeScript = "ts"
	LanguageShorthandPython     = "py"
)

type TemplateInfo struct {
	Name        string
	Description string
	Languages   []Language
}

var Templates = map[string]TemplateInfo{
	"sample-app": {
		Name:        "Sample App",
		Description: "Implements basic Kernel apps",
		Languages:   []Language{LanguageTypeScript, LanguagePython},
	},
}

// SupportedLanguages returns a list of all supported languages
var SupportedLanguages = []string{
	LanguageTypeScript,
	LanguagePython,
}

// GetSupportedTemplates returns a list of all supported template names
func GetSupportedTemplates() []string {
	templates := make([]string, 0, len(Templates))
	for tn := range Templates {
		templates = append(templates, tn)
	}
	return templates
}

// Helper to normalize language input (handle shorthand)
func NormalizeLanguage(language string) string {
	switch language {
	case LanguageShorthandTypeScript:
		return LanguageTypeScript
	case LanguageShorthandPython:
		return LanguagePython
	default:
		return language
	}
}
