package cmd

import (
	"fmt"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Long:  "Commands for creating new Kernel applications",
	RunE:  runCreateApp,
}

func init() {
	createCmd.Flags().String("name", "", "Name of the application")
	createCmd.Flags().String("language", "", "Language of the application")
	createCmd.Flags().String("template", "", "Template to use for the application")
}

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

// promptForAppName prompts the user for the application name if not provided
func promptForAppName(providedAppName string) (string, error) {
	if providedAppName != "" {
		return providedAppName, nil
	}

	var appName string
	prompt := &survey.Input{
		Message: "What is the name of your project?",
		Default: defaultAppName,
	}

	err := survey.AskOne(prompt, &appName, survey.WithValidator(projectNameValidator))
	if err != nil {
		return "", err
	}

	return appName, nil
}

func runCreateApp(cmd *cobra.Command, args []string) error {
	providedAppName, _ := cmd.Flags().GetString("name")
	language, _ := cmd.Flags().GetString("language")
	template, _ := cmd.Flags().GetString("template")

	// Prompt for app name if not provided
	appName, err := promptForAppName(providedAppName)
	if err != nil {
		return fmt.Errorf("failed to get app name: %w", err)
	}

	fmt.Printf("Creating application '%s' with language '%s' and template '%s'...\n", appName, language, template)

	// TODO: prompt the user for the language of the app, suggest a default language (typescript)
	// TODO: prompt the user for the template of the app, suggest a default template (sample-app)

	// TODO: create the project structure

	// print "Creating a new TypeScript Sample App" or similar. Essentially the language and template name combined.

	/*
		Print the following:
				✔ TypeScript environment set up successfully

				🎉 Kernel app created successfully!

				Next steps:
				  brew install onkernel/tap/kernel
				  cd my-kernel-app
				  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
				  kernel deploy index.ts
				  kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'
				  # Do this in a separate tab
				  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
				  kernel logs ts-basic --follow
	*/

	return nil
}
