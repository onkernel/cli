package create

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/AlecAivazis/survey/v2"
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

	var appName string
	prompt := &survey.Input{
		Message: AppNamePrompt,
		Default: DefaultAppName,
	}

	if err := survey.AskOne(prompt, &appName, survey.WithValidator(validateAppName)); err != nil {
		return "", err
	}

	return appName, nil
}

func handleLangugePrompt() (string, error) {
	var l string
	languagePrompt := &survey.Select{
		Message: LanguagePrompt,
		Options: SupportedLanguages,
	}
	if err := survey.AskOne(languagePrompt, &l); err != nil {
		return "", err
	}
	return l, nil
}

func PromptForLanguage(providedLanguage string) (string, error) {
	if providedLanguage == "" {
		return handleLangugePrompt()
	}

	l := NormalizeLanguage(providedLanguage)
	if slices.Contains(SupportedLanguages, l) {
		return l, nil
	}

	return handleLangugePrompt()
}

// TODO: add validation for template
func PromptForTemplate(providedTemplate string) (string, error) {
	if providedTemplate != "" {
		return providedTemplate, nil
	}

	var template string
	templatePrompt := &survey.Select{
		Message: TemplatePrompt,
		Options: GetSupportedTemplates(),
	}
	if err := survey.AskOne(templatePrompt, &template); err != nil {
		return "", err
	}
	return template, nil
}
