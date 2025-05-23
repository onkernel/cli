package cmd

import (
	"fmt"
	"time"

	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <app_name>",
	Short: "Show logs for a Kernel application",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().String("version", "latest", "Specify a version of the app (default: latest)")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow logs in real-time (stream continuously)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	client := kernel.NewClient()

	appName := args[0]
	version, _ := cmd.Flags().GetString("version")
	follow, _ := cmd.Flags().GetBool("follow")
	if version == "" {
		version = "latest"
	}

	params := kernel.AppListParams{
		AppName: kernel.Opt(appName),
		Version: kernel.Opt(version),
	}
	apps, err := client.Apps.List(cmd.Context(), params)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}
	if apps == nil || len(*apps) == 0 {
		return fmt.Errorf("app \"%s\" not found", appName)
	}
	if len(*apps) > 1 {
		return fmt.Errorf("multiple apps found for \"%s\", please specify a version", appName)
	}
	app := (*apps)[0]

	pterm.Info.Printf("Streaming logs for app \"%s\" (version: %s, id: %s)...\n", appName, version, app.ID)
	if follow {
		pterm.Info.Println("Press Ctrl+C to exit")
	} else {
		pterm.Info.Println("Showing recent logs (timeout after 3s with no events)")
	}

	stream := client.Apps.Deployments.FollowStreaming(cmd.Context(), app.ID)

	// Handle follow vs non-follow mode
	if follow {
		// Keep streaming indefinitely
		for stream.Next() {
			data := stream.Current()
			switch data.Event {
			case "log":
				fmt.Println(data.AsLog().Message)
			}
		}
	} else {
		// Exit after 3 seconds of no activity
		timeout := time.NewTimer(3 * time.Second)
		defer timeout.Stop()

		done := false
		for !done {
			// Create a channel for the Next() operation
			nextCh := make(chan bool, 1)

			// Start a goroutine to check for the next event
			go func() {
				hasNext := stream.Next()
				nextCh <- hasNext
			}()

			// Wait for either next event or timeout
			select {
			case hasNext := <-nextCh:
				if !hasNext {
					done = true
				} else {
					// Got an event, display it and reset timer
					data := stream.Current()
					switch data.Event {
					case "log":
						fmt.Println(data.AsLog().Message)
					}
					timeout.Reset(3 * time.Second)
				}
			case <-timeout.C:
				// No events for 3 seconds, we're done
				done = true
				stream.Close()
			}
		}
	}

	if stream.Err() != nil {
		return fmt.Errorf("failed to follow streaming: %w", stream.Err())
	}
	return nil
}
