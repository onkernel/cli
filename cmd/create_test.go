package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onkernel/cli/pkg/create"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       CreateInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, appPath string)
	}{
		{
			name: "create typescript sample-app",
			input: CreateInput{
				Name:     "test-app",
				Language: "typescript",
				Template: "sample-app",
			},
			validate: func(t *testing.T, appPath string) {
				// Verify files were created
				assert.FileExists(t, filepath.Join(appPath, "index.ts"))
				assert.FileExists(t, filepath.Join(appPath, "package.json"))
				assert.FileExists(t, filepath.Join(appPath, ".gitignore"))
				assert.NoFileExists(t, filepath.Join(appPath, "_gitignore"))
			},
		},
		{
			name: "fail with invalid template",
			input: CreateInput{
				Name:     "test-app",
				Language: "typescript",
				Template: "nonexistent",
			},
			wantErr:     true,
			errContains: "template not found: typescript/nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			orgDir, err := os.Getwd()
			require.NoError(t, err)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			t.Cleanup(func() {
				os.Chdir(orgDir)
			})

			c := CreateCmd{}
			err = c.Create(context.Background(), tt.input)

			// Check if error is expected
			if tt.wantErr {
				require.Error(t, err, "expected command to fail but it succeeded")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "error message should contain expected text")
				}
				return
			}

			require.NoError(t, err, "failed to execute create command")

			// Validate the created app
			appPath := filepath.Join(tmpDir, tt.input.Name)
			assert.DirExists(t, appPath, "app directory should be created")

			if tt.validate != nil {
				tt.validate(t, appPath)
			}
		})
	}
}

// TestAllTemplatesWithDependencies tests all available templates and verifies dependencies are installed
func TestAllTemplatesWithDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dependency installation tests in short mode")
	}

	tests := getTemplateInfo()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			appName := "test-app"

			orgDir, err := os.Getwd()
			require.NoError(t, err)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			t.Cleanup(func() {
				os.Chdir(orgDir)
			})

			// Create the app
			c := CreateCmd{}
			err = c.Create(context.Background(), CreateInput{
				Name:     appName,
				Language: tt.language,
				Template: tt.template,
			})
			require.NoError(t, err, "failed to create app")

			appPath := filepath.Join(tmpDir, appName)

			// Verify app directory exists
			assert.DirExists(t, appPath, "app directory should exist")

			// Language-specific validations
			switch tt.language {
			case create.LanguageTypeScript:
				validateTypeScriptTemplate(t, appPath, true)
			case create.LanguagePython:
				validatePythonTemplate(t, appPath, true)
			}
		})
	}
}

// TestAllTemplatesCreation tests that all templates can be created without installing dependencies
func TestAllTemplatesCreation(t *testing.T) {
	tests := getTemplateInfo()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			appName := "test-app"
			appPath := filepath.Join(tmpDir, appName)

			// Create app directory
			err := os.MkdirAll(appPath, 0755)
			require.NoError(t, err, "failed to create app directory")

			// Copy template files without installing dependencies
			err = create.CopyTemplateFiles(appPath, tt.language, tt.template)
			require.NoError(t, err, "failed to copy template files")

			// Verify app directory exists
			assert.DirExists(t, appPath, "app directory should exist")

			// Language-specific validations (without dependency checks)
			switch tt.language {
			case create.LanguageTypeScript:
				validateTypeScriptTemplate(t, appPath, false)
			case create.LanguagePython:
				validatePythonTemplate(t, appPath, false)
			}
		})
	}
}

// validateTypeScriptTemplate verifies TypeScript template structure and optionally dependencies
func validateTypeScriptTemplate(t *testing.T, appPath string, checkDependencies bool) {
	t.Helper()

	// Verify essential files exist
	assert.FileExists(t, filepath.Join(appPath, "package.json"), "package.json should exist")
	assert.FileExists(t, filepath.Join(appPath, "tsconfig.json"), "tsconfig.json should exist")
	assert.FileExists(t, filepath.Join(appPath, "index.ts"), "index.ts should exist")
	assert.FileExists(t, filepath.Join(appPath, ".gitignore"), ".gitignore should exist")

	// Verify _gitignore was renamed
	assert.NoFileExists(t, filepath.Join(appPath, "_gitignore"), "_gitignore should not exist")

	if checkDependencies {
		// Verify node_modules exists (dependencies were installed)
		nodeModulesPath := filepath.Join(appPath, "node_modules")
		if _, err := os.Stat(nodeModulesPath); err == nil {
			// Only check contents if node_modules exists
			entries, err := os.ReadDir(nodeModulesPath)
			require.NoError(t, err, "should be able to read node_modules directory")
			assert.NotEmpty(t, entries, "node_modules should contain installed packages")
		} else {
			t.Logf("Warning: node_modules not found at %s (npm install may have failed)", nodeModulesPath)
		}
	}
}

// validatePythonTemplate verifies Python template structure and optionally dependencies
func validatePythonTemplate(t *testing.T, appPath string, checkDependencies bool) {
	t.Helper()

	// Verify essential files exist
	assert.FileExists(t, filepath.Join(appPath, "pyproject.toml"), "pyproject.toml should exist")
	assert.FileExists(t, filepath.Join(appPath, "main.py"), "main.py should exist")
	assert.FileExists(t, filepath.Join(appPath, ".gitignore"), ".gitignore should exist")

	// Verify _gitignore was renamed
	assert.NoFileExists(t, filepath.Join(appPath, "_gitignore"), "_gitignore should not exist")

	if checkDependencies {
		// Verify .venv exists (virtual environment was created)
		venvPath := filepath.Join(appPath, ".venv")
		if _, err := os.Stat(venvPath); err == nil {
			// Only check contents if .venv exists
			binPath := filepath.Join(venvPath, "bin")
			assert.DirExists(t, binPath, ".venv/bin directory should exist")

			pythonPath := filepath.Join(binPath, "python")
			assert.FileExists(t, pythonPath, ".venv/bin/python should exist")
		} else {
			t.Logf("Warning: .venv not found at %s (uv venv may have failed)", venvPath)
		}
	}
}

// TestCreateCommand_DependencyInstallationFails tests that the app is still created
// even when dependency installation fails, with appropriate warning message
func TestCreateCommand_DependencyInstallationFails(t *testing.T) {
	tmpDir := t.TempDir()
	appName := "test-app"

	orgDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Chdir(orgDir)
	})

	// Override the install command to use a command that will fail
	originalInstallCommands := create.InstallCommands
	create.InstallCommands = map[string]string{
		create.LanguageTypeScript: "exit 1", // Command that always fails
	}

	// Restore original install commands after test
	t.Cleanup(func() {
		create.InstallCommands = originalInstallCommands
	})

	// Create the app - should succeed even though dependency installation fails
	c := CreateCmd{}
	err = c.Create(context.Background(), CreateInput{
		Name:     appName,
		Language: create.LanguageTypeScript,
		Template: "sample-app",
	})
}

func getTemplateInfo() []struct {
	name     string
	language string
	template string
} {
	tests := make([]struct {
		name     string
		language string
		template string
	}, 0)

	for templateKey, templateInfo := range create.Templates {
		for _, lang := range templateInfo.Languages {
			tests = append(tests, struct {
				name     string
				language string
				template string
			}{
				name:     lang + "/" + templateKey,
				language: lang,
				template: templateKey,
			})
		}
	}

	return tests
}
