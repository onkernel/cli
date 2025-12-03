package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onkernel/cli/pkg/create"
	"github.com/pterm/pterm"
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

	// Get absolute path for the app directory
	appPath, err := filepath.Abs(appName)
	if err != nil {
		return fmt.Errorf("failed to resolve app path: %w", err)
	}

	// Check if directory already exists
	if _, err := os.Stat(appPath); err == nil {
		return fmt.Errorf("directory %s already exists", appName)
	}

	// Create the app directory
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	fmt.Printf("\nCreating a new %s %s\n\n", language, template)

	spinner, _ := pterm.DefaultSpinner.Start("Copying template files...")

	if err := create.CopyTemplateFiles(appPath, language, template); err != nil {
		spinner.Fail("Failed to copy template files")
		return fmt.Errorf("failed to copy template files: %w", err)
	}
	spinner.Success("✔ TypeScript environment set up successfully")

	nextSteps := fmt.Sprintf(`Next steps:
  brew install onkernel/tap/kernel
  cd %s
  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
  kernel deploy index.ts
  kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'
  # Do this in a separate tab
  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
  kernel logs ts-basic --follow
`, appName)

	pterm.Success.Println("🎉 Kernel app created successfully!")
	pterm.Println()
	pterm.FgYellow.Println(nextSteps)

	return nil
}
