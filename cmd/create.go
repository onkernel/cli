package cmd

import (
	"fmt"

	"github.com/onkernel/cli/pkg/create"
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

func runCreateApp(cmd *cobra.Command, args []string) error {
	appName, _ := cmd.Flags().GetString("name")
	language, _ := cmd.Flags().GetString("language")
	template, _ := cmd.Flags().GetString("template")

	appName, err := create.PromptForAppName(appName)
	if err != nil {
		return fmt.Errorf("failed to get app name: %w", err)
	}

	language, err = create.PromptForLanguage(language)
	if err != nil {
		return fmt.Errorf("failed to get language: %w", err)
	}

	template, err = create.PromptForTemplate(template)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	fmt.Printf("Creating application '%s' with language '%s' and template '%s'...\n", appName, language, template)

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
