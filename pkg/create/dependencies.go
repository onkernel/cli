package create

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/pterm/pterm"
)

// InstallDependencies sets up project dependencies based on language
func InstallDependencies(appPath string, language string) error {
	installCommand, ok := InstallCommands[language]
	if !ok {
		return fmt.Errorf("unsupported language: %s", language)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Setting up %s environment...", language))

	cmd := exec.Command("sh", "-c", installCommand)
	cmd.Dir = appPath

	if err := cmd.Run(); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to set up %s environment", language))
		pterm.Error.Printf("Error: %v\n", err)

		pterm.FgYellow.Println("\nPlease install dependencies manually:")
		switch language {
		case LanguageTypeScript:
			pterm.Println(fmt.Sprintf("  cd %s", filepath.Base(appPath)))
			pterm.Println("  npm install")
		case LanguagePython:
			pterm.Println(fmt.Sprintf("  cd %s", filepath.Base(appPath)))
			pterm.Println("  uv venv && source .venv/bin/activate && uv sync")
		}

		return nil
	}

	spinner.Success(fmt.Sprintf("%s environment set up successfully", language))
	return nil
}
