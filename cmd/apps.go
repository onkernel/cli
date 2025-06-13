package cmd

import (
	"strings"

	"github.com/onkernel/cli/pkg/util"
	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage deployed applications",
	Long:  "Commands for managing deployed Kernel applications",
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed application versions",
	RunE:  runAppsList,
}

func init() {
	appsCmd.AddCommand(appsListCmd)

	// Add optional filters
	appsListCmd.Flags().String("name", "", "Filter by application name")
	appsListCmd.Flags().String("version", "", "Filter by version label")
}

func runAppsList(cmd *cobra.Command, args []string) error {
	client := util.NewClient()

	appName, _ := cmd.Flags().GetString("name")
	version, _ := cmd.Flags().GetString("version")

	pterm.Debug.Println("Fetching deployed applications...")

	params := kernel.AppListParams{}
	if appName != "" {
		params.AppName = kernel.Opt(appName)
	}
	if version != "" {
		params.Version = kernel.Opt(version)
	}

	apps, err := client.Apps.List(cmd.Context(), params)
	if err != nil {
		pterm.Error.Printf("Failed to list applications: %v\n", err)
		return nil
	}

	if apps == nil || len(*apps) == 0 {
		pterm.Info.Println("No applications found")
		return nil
	}

	// Prepare table data
	tableData := pterm.TableData{
		{"App Name", "Version", "App Version ID", "Region", "Env Vars"},
	}

	for _, app := range *apps {
		// Format env vars
		envVarsStr := "-"
		if app.EnvVars != nil && len(app.EnvVars) > 0 {
			var envKeys []string
			for k := range app.EnvVars {
				envKeys = append(envKeys, k)
			}
			envVarsStr = strings.Join(envKeys, ", ")
			if len(envVarsStr) > 50 {
				envVarsStr = envVarsStr[:47] + "..."
			}
		}

		tableData = append(tableData, []string{
			app.AppName,
			app.Version,
			app.ID,
			app.Region,
			envVarsStr,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}
