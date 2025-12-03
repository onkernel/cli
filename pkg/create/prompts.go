package create

import (
	"fmt"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
)

const defaultAppName = "my-kernel-app"

// projectNameValidator ensures the project name is safe for file systems and package managers.
func projectNameValidator(val any) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid input type")
	}

	// Project name must be non-empty
	if len(str) == 0 {
		return fmt.Errorf("project name cannot be empty")
	}

	// Validate project name: only letters, numbers, underscores, and hyphens
	// This regex prevents special characters that might break shell commands or filesystem paths.
	matched, err := regexp.MatchString(`^[A-Za-z\-_\d]+$`, str)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("project name may only include letters, numbers, underscores, and hyphens")
	}
	return nil
}

// PromptForAppName prompts the user for the application name if not provided
func PromptForAppName(providedAppName string) (string, error) {
	if providedAppName != "" {
		return providedAppName, nil
	}

	var appName string
	prompt := &survey.Input{
		Message: "What is the name of your project?",
		Default: defaultAppName,
	}

	if err := survey.AskOne(prompt, &appName, survey.WithValidator(projectNameValidator)); err != nil {
		return "", err
	}

	return appName, nil
}

func PromptForLanguage(providedLanguage string) (string, error) {
	if providedLanguage != "" {
		return providedLanguage, nil
	}
	var language string
	languagePrompt := &survey.Select{
		Message: "Choose a programming language:",
		// TODO: create constants so that more languages can be added later
		Options: []string{"typescript", "python"},
		Default: "typescript",
	}
	if err := survey.AskOne(languagePrompt, &language); err != nil {
		return "", err
	}
	return language, nil
}

func PromptForTemplate(providedTemplate string) (string, error) {
	if providedTemplate != "" {
		return providedTemplate, nil
	}

	var template string
	templatePrompt := &survey.Select{
		Message: "Choose a template:",
		Options: []string{"sample-app"},
		Default: "sample-app",
	}
	if err := survey.AskOne(templatePrompt, &template); err != nil {
		return "", err
	}
	return template, nil
}
