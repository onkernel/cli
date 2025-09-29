package proxies

import (
	"github.com/onkernel/kernel-go-sdk"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// GetKernelClient retrieves the kernel client from the command context
func GetKernelClient(cmd *cobra.Command) kernel.Client {
	// Use the raw string key to match what cmd package uses
	// This avoids type mismatch between different contextKey types
	return cmd.Context().Value("kernel_client").(kernel.Client)
}

// PrintTableNoPad prints a table without padding (delegating to cmd package)
func PrintTableNoPad(data pterm.TableData, withRowSeparators bool) {
	table := pterm.DefaultTable.WithHasHeader().WithData(data)
	if withRowSeparators {
		table = table.WithRowSeparator("-")
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(data).WithRowSeparator("-").Render()
}
