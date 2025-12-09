package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onkernel/cli/pkg/create"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type CreateInput struct {
	Name     string
	Language string
	Template string
}

// CreateCmd is a cobra-independent command handler for create operations
type CreateCmd struct{}

// Create executes the creating a new Kernel app logic
func (c CreateCmd) Create(ctx context.Context, ci CreateInput) error {
	appPath, err := filepath.Abs(ci.Name)
	if err != nil {
		return fmt.Errorf("failed to resolve app path: %w", err)
	}

	// TODO: handle overwrite gracefully (prompt user)
	// Check if directory already exists
	if _, err := os.Stat(appPath); err == nil {
		return fmt.Errorf("directory %s already exists", ci.Name)
	}

	if err := os.MkdirAll(appPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	pterm.Println(fmt.Sprintf("\nCreating a new %s %s\n", ci.Language, ci.Template))

	spinner, _ := pterm.DefaultSpinner.Start("Copying template files...")

	if err := create.CopyTemplateFiles(appPath, ci.Language, ci.Template); err != nil {
		spinner.Fail("Failed to copy template files")
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	ok, _ := create.InstallDependencies(appPath, ci.Language)
	if !ok {
		pterm.Warning.Println("Failed to install dependencies. Please install them manually:")
		switch ci.Language {
		case create.LanguageTypeScript:
			pterm.Println(fmt.Sprintf("  cd %s", ci.Name))
			pterm.Println("  npm install")
		case create.LanguagePython:
			pterm.Println(fmt.Sprintf("  cd %s", ci.Name))
			pterm.Println("  uv venv && source .venv/bin/activate && uv sync")
		}
		pterm.Println()
	} else {
		spinner.Success(fmt.Sprintf("✔ %s environment set up successfully", ci.Language))
	}

	pterm.Success.Println("🎉 Kernel app created successfully!")
	pterm.Println()

	nextSteps := fmt.Sprintf(`Next steps:
  brew install onkernel/tap/kernel
  cd %s
  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
  kernel deploy index.ts
  kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'
`, ci.Name)

	pterm.FgYellow.Println(nextSteps)

	return nil
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Long:  "Commands for creating new Kernel applications",
	RunE:  runCreateApp,
}

func init() {
	createCmd.Flags().StringP("name", "n", "", "Name of the application")
	createCmd.Flags().StringP("language", "l", "", "Language of the application")
	createCmd.Flags().StringP("template", "t", "", "Template to use for the application")
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

	template, err = create.PromptForTemplate(template, language)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	c := CreateCmd{}
	return c.Create(cmd.Context(), CreateInput{
		Name:     appName,
		Language: language,
		Template: template,
	})
}
