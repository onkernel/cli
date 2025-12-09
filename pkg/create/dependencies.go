package create

import (
	"fmt"
	"os/exec"

	"github.com/pterm/pterm"
)

// InstallDependencies sets up project dependencies based on language
func InstallDependencies(appPath string, language string) (bool, error) {
	installCommand, ok := InstallCommands[language]
	if !ok {
		return false, fmt.Errorf("unsupported language: %s", language)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Setting up %s environment...", language))

	cmd := exec.Command("sh", "-c", installCommand)
	cmd.Dir = appPath

	if err := cmd.Run(); err != nil {
		spinner.Stop()
		return false, nil
	}

	spinner.Success(fmt.Sprintf("%s environment set up successfully", language))
	return true, nil
}
