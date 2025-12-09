package create

const (
	DefaultAppName = "my-kernel-app"
	AppNamePrompt  = "What is the name of your project?"
	LanguagePrompt = "Choose a programming language:"
	TemplatePrompt = "Select a template:"
)

const (
	LanguageTypeScript          = "typescript"
	LanguagePython              = "python"
	LanguageShorthandTypeScript = "ts"
	LanguageShorthandPython     = "py"
)

var InstallCommands = map[string]string{
	LanguageTypeScript: "npm install",
	LanguagePython:     "uv venv",
}

// SupportedLanguages returns a list of all supported languages
var SupportedLanguages = []string{
	LanguageTypeScript,
	LanguagePython,
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
