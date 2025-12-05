package create

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/AlecAivazis/survey/v2"
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

	pterm.Warning.Printfln("Language '%s' not found. Please select from available languages.\n", providedLanguage)
	return handleLangugePrompt()
}

func handleTemplatePrompt(templateKVs TemplateKeyValues) (string, error) {
	var selectedValue string
	templatePrompt := &survey.Select{
		Message: TemplatePrompt,
		Options: templateKVs.GetTemplateDisplayValues(),
	}
	if err := survey.AskOne(templatePrompt, &selectedValue); err != nil {
		return "", err
	}

	return templateKVs.GetTemplateKeyFromValue(selectedValue)
}

func PromptForTemplate(providedTemplate string, providedLanguage string) (string, error) {
	templateKVs := GetSupportedTemplatesForLanguage(NormalizeLanguage(providedLanguage))

	if providedTemplate == "" {
		return handleTemplatePrompt(templateKVs)
	}

	if templateKVs.ContainsKey(providedTemplate) {
		return providedTemplate, nil
	}

	pterm.Warning.Printfln("Template '%s' not found. Please select from available templates.\n", providedTemplate)
	return handleTemplatePrompt(templateKVs)
}
