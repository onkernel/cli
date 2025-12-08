package create

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/pterm/pterm"
)

func validateAppName(val any) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid input type")
	}

	if len(str) == 0 {
		return fmt.Errorf("project name cannot be empty")
	}

	// Validate project name: only letters, numbers, underscores, and hyphens
	matched, err := regexp.MatchString(`^[A-Za-z\-_\d]+$`, str)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("project name may only include letters, numbers, underscores, and hyphens")
	}
	return nil
}

func PromptForAppName(providedAppName string) (string, error) {
	if providedAppName != "" {
		return providedAppName, nil
	}

	promptText := fmt.Sprintf("%s (default: %s)", AppNamePrompt, DefaultAppName)
	appName, err := pterm.DefaultInteractiveTextInput.
		WithDefaultText(promptText).
		Show()
	if err != nil {
		return "", err
	}

	// Use default if user just pressed enter without typing anything
	if appName == "" {
		appName = DefaultAppName
	}

	// Validate the app name
	if err := validateAppName(appName); err != nil {
		return "", err
	}

	return appName, nil
}

func handleLanguagePrompt() (string, error) {
	l, err := pterm.DefaultInteractiveSelect.
		WithOptions(SupportedLanguages).
		WithDefaultText(LanguagePrompt).
		Show()
	if err != nil {
		return "", err
	}
	return l, nil
}

func PromptForLanguage(providedLanguage string) (string, error) {
	if providedLanguage == "" {
		return handleLanguagePrompt()
	}

	l := NormalizeLanguage(providedLanguage)
	if slices.Contains(SupportedLanguages, l) {
		return l, nil
	}

	return handleLanguagePrompt()
}

// TODO: add validation for template
func PromptForTemplate(providedTemplate string) (string, error) {
	if providedTemplate != "" {
		return providedTemplate, nil
	}

	template, err := pterm.DefaultInteractiveSelect.
		WithOptions(GetSupportedTemplates()).
		WithDefaultText(TemplatePrompt).
		Show()
	if err != nil {
		return "", err
	}
	return template, nil
}
