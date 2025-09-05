package cmd

import (
	"strings"
	"time"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage deployed applications",
	Long:  "Commands for managing deployed Kernel applications",
}

// --- app list subcommand
var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed application versions",
	RunE:  runAppList,
}

// --- app history subcommand (scaffold)
var appHistoryCmd = &cobra.Command{
	Use:   "history <app_name>",
	Short: "Show deployment history for an application",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppHistory,
}

func init() {
	// register subcommands under app
	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appHistoryCmd)

	// Add optional filters for list
	appListCmd.Flags().String("name", "", "Filter by application name")
	appListCmd.Flags().String("version", "", "Filter by version label")

	// Limit rows returned for app history (0 = all)
	appHistoryCmd.Flags().Int("limit", 100, "Max rows to return (default 100)")
}

func runAppList(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
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
		{"App Name", "Version", "App Version ID", "Region", "Actions", "Env Vars"},
	}

	for _, app := range *apps {
		// Format env vars
		envVarsStr := "-"
		if len(app.EnvVars) > 0 {
			envVarsStr = strings.Join(lo.Keys(app.EnvVars), ", ")
			if len(envVarsStr) > 50 {
				envVarsStr = envVarsStr[:47] + "..."
			}
		}

		actionsStr := "-"
		if len(app.Actions) > 0 {
			actionsStr = strings.Join(lo.Map(app.Actions, func(a kernel.AppAction, _ int) string {
				return a.Name
			}), ", ")
			if len(actionsStr) > 50 {
				actionsStr = actionsStr[:47] + "..."
			}
		}

		tableData = append(tableData, []string{
			app.AppName,
			app.Version,
			app.ID,
			string(app.Region),
			actionsStr,
			envVarsStr,
		})
	}

	printTableNoPad(tableData, true)
	return nil
}

func runAppHistory(cmd *cobra.Command, args []string) error {
	client := getKernelClient(cmd)
	appName := args[0]
	lim, _ := cmd.Flags().GetInt("limit")

	pterm.Debug.Printf("Fetching deployment history for app '%s'...\n", appName)

	params := kernel.DeploymentListParams{}
	if appName != "" {
		params.AppName = kernel.Opt(appName)
	}

	page, err := client.Deployments.List(cmd.Context(), params)
	if err != nil {
		pterm.Error.Printf("Failed to list deployments: %v\n", err)
		return nil
	}

	if page == nil || len(page.Items) == 0 {
		pterm.Info.Println("No deployments found for this application")
		return nil
	}

	tableData := pterm.TableData{
		{"Deployment ID", "Created At", "Region", "Status", "Entrypoint", "Reason"},
	}

	rows := 0
	stop := false
	for page != nil && !stop {
		for _, dep := range page.Items {
			created := dep.CreatedAt.Format(time.RFC3339)
			status := string(dep.Status)

			tableData = append(tableData, []string{
				dep.ID,
				created,
				string(dep.Region),
				status,
				dep.EntrypointRelPath,
				dep.StatusReason,
			})

			rows++
			if lim > 0 && rows >= lim {
				stop = true
				break
			}
		}
		if stop {
			break
		}
		page, err = page.GetNextPage()
		if err != nil {
			pterm.Error.Printf("Failed to fetch next page: %v\n", err)
			break
		}
	}

	printTableNoPad(tableData, true)
	return nil
}
