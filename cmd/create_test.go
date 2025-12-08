package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
