package create

import (
	"fmt"
	"os/exec"

	"github.com/pterm/pterm"
)

// InstallDependencies sets up project dependencies based on language
func InstallDependencies(appName string, appPath string, language string) (string, error) {
	installCommand, ok := InstallCommands[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	requiredTool := RequiredTools[language]
	if requiredTool != "" && !RequiredTools.CheckToolAvailable(language) {
		return getNextStepsWithToolInstall(appName, language, requiredTool), nil
	}

	spinner, _ := pterm.DefaultSpinner.Start(pterm.Sprintf("Setting up %s environment...", language))

	cmd := exec.Command("sh", "-c", installCommand)
	cmd.Dir = appPath

	if err := cmd.Run(); err != nil {
		spinner.Stop()
		pterm.Warning.Println("Failed to install dependencies. Please install them manually:")
		switch language {
		case LanguageTypeScript:
			pterm.Printfln("  cd %s", appName)
			pterm.Printfln("  pnpm install")
		case LanguagePython:
			pterm.Printfln("  cd %s", appName)
			pterm.Println("  uv venv && source .venv/bin/activate && uv sync")
		}
		pterm.Println()
		return getNextStepsStandard(appName), nil
	}

	spinner.Success(pterm.Sprintf("✔ %s environment set up successfully", language))

	return getNextStepsStandard(appName), nil
}

// getNextStepsWithToolInstall returns next steps message including tool installation
func getNextStepsWithToolInstall(appName string, language string, requiredTool string) string {
	pterm.Warning.Printfln(" %s is not installed or not in PATH", requiredTool)

	switch language {
	case LanguageTypeScript:
		return pterm.FgYellow.Sprintf(`Next steps:
  # Install pnpm (choose one):
  npm install -g pnpm
  # or: brew install pnpm
  # or: curl -fsSL https://get.pnpm.io/install.sh | sh -

  # Then install dependencies:
  cd %s
  pnpm install

  # Deploy your app:
  brew install onkernel/tap/kernel
  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
  kernel deploy index.ts
  kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'
`, appName)
	case LanguagePython:
		return pterm.FgYellow.Sprintf(`Next steps:
  # Install uv (choose one):
  curl -LsSf https://astral.sh/uv/install.sh | sh
  # or: brew install uv
  # or: pipx install uv

  # Then set up your environment:
  cd %s
  uv venv && source .venv/bin/activate && uv sync

  # Deploy your app:
  brew install onkernel/tap/kernel
  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
  kernel deploy index.py
  kernel invoke py-basic get-page-title --payload '{"url": "https://www.google.com"}'
`, appName)
	default:
		return ""
	}
}

// getNextStepsStandard returns standard next steps message
func getNextStepsStandard(appName string) string {
	return pterm.FgYellow.Sprintf(`Next steps:
  brew install onkernel/tap/kernel
  cd %s
  kernel login  # or: export KERNEL_API_KEY=<YOUR_API_KEY>
  kernel deploy index.ts
  kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'
`, appName)
}
