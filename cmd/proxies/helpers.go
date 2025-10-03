package proxies

import (
	"github.com/pterm/pterm"
)

// PrintTableNoPad prints a table without padding
func PrintTableNoPad(data pterm.TableData, withRowSeparators bool) {
	table := pterm.DefaultTable.WithHasHeader().WithData(data)
	if withRowSeparators {
		table = table.WithRowSeparator("-")
	}
	_ = table.Render()
}
