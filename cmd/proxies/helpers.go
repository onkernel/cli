package proxies

import (
	"unicode/utf8"

	"github.com/pterm/pterm"
)

// PrintTableNoPad prints a table with intelligent column truncation to prevent wrapping.
// It detects terminal width and truncates cells that would cause line wrapping,
// adding "..." to indicate truncation.
func PrintTableNoPad(data pterm.TableData, withRowSeparators bool) {
	if len(data) == 0 {
		return
	}

	// Get terminal width
	termWidth := pterm.GetTerminalWidth()
	if termWidth <= 0 {
		termWidth = 80 // fallback
	}

	// Truncate data to fit terminal width
	truncatedData := truncateTableData(data, termWidth)

	table := pterm.DefaultTable.WithHasHeader().WithData(truncatedData)
	if withRowSeparators {
		table = table.WithRowSeparator("-")
	}
	_ = table.Render()
}

// truncateTableData intelligently truncates table cells to fit within terminal width
func truncateTableData(data pterm.TableData, termWidth int) pterm.TableData {
	if len(data) == 0 {
		return data
	}

	numCols := len(data[0])
	if numCols == 0 {
		return data
	}

	// Calculate separator space: " | " between each column (3 chars per separator)
	separatorSpace := (numCols - 1) * 3

	// Define minimum column widths (these are the bare minimum before aggressive truncation)
	minWidths := make([]int, numCols)
	for i := 0; i < numCols; i++ {
		minWidths[i] = 8 // minimum 8 chars per column
	}

	// Calculate natural widths (what each column would want)
	naturalWidths := make([]int, numCols)
	for colIdx := 0; colIdx < numCols; colIdx++ {
		maxWidth := 0
		for _, row := range data {
			if colIdx < len(row) {
				cellWidth := utf8.RuneCountInString(row[colIdx])
				if cellWidth > maxWidth {
					maxWidth = cellWidth
				}
			}
		}
		naturalWidths[colIdx] = maxWidth
	}

	// Calculate available space for content
	availableWidth := termWidth - separatorSpace - 2 // -2 for margins

	// Distribute width among columns
	columnWidths := distributeColumnWidths(naturalWidths, minWidths, availableWidth)

	// Truncate cells based on calculated widths
	result := make(pterm.TableData, len(data))
	for rowIdx, row := range data {
		result[rowIdx] = make([]string, len(row))
		for colIdx, cell := range row {
			if colIdx < len(columnWidths) {
				result[rowIdx][colIdx] = truncateCell(cell, columnWidths[colIdx])
			} else {
				result[rowIdx][colIdx] = cell
			}
		}
	}

	return result
}

// distributeColumnWidths calculates optimal width for each column
// The first column (ID) is always given its full natural width and never truncated
func distributeColumnWidths(naturalWidths, minWidths []int, availableWidth int) []int {
	numCols := len(naturalWidths)
	result := make([]int, numCols)

	// Start with natural widths
	copy(result, naturalWidths)

	// Calculate total natural width needed
	totalNatural := 0
	for _, w := range naturalWidths {
		totalNatural += w
	}

	// If natural widths fit, use them
	if totalNatural <= availableWidth {
		return result
	}

	// Priority: Always give the ID column (first column) its full natural width
	if numCols > 0 {
		result[0] = naturalWidths[0]
		availableWidth -= naturalWidths[0]
	}

	// Now distribute remaining width among other columns (excluding ID)
	if numCols <= 1 {
		return result
	}

	// Calculate needs for non-ID columns
	otherCols := numCols - 1
	totalMinForOthers := 0
	totalNaturalForOthers := 0
	for i := 1; i < numCols; i++ {
		totalMinForOthers += minWidths[i]
		totalNaturalForOthers += naturalWidths[i]
	}

	if totalNaturalForOthers <= availableWidth {
		// Other columns fit naturally
		for i := 1; i < numCols; i++ {
			result[i] = naturalWidths[i]
		}
		return result
	}

	if totalMinForOthers > availableWidth {
		// Even minimums don't fit for other columns, distribute equally
		for i := 1; i < numCols; i++ {
			result[i] = availableWidth / otherCols
			if result[i] < 5 {
				result[i] = 5 // absolute minimum
			}
		}
		return result
	}

	// Give other columns minimum, then distribute remainder proportionally
	remainingWidth := availableWidth - totalMinForOthers
	remainingNeed := totalNaturalForOthers - totalMinForOthers

	for i := 1; i < numCols; i++ {
		result[i] = minWidths[i]
		if remainingNeed > 0 {
			additionalNeed := naturalWidths[i] - minWidths[i]
			additionalGrant := (additionalNeed * remainingWidth) / remainingNeed
			result[i] += additionalGrant
		}
	}

	return result
}

// truncateCell truncates a cell to maxWidth, adding "..." if truncated
func truncateCell(cell string, maxWidth int) string {
	cellWidth := utf8.RuneCountInString(cell)
	if cellWidth <= maxWidth {
		return cell
	}

	if maxWidth <= 3 {
		// Too narrow for "...", just truncate
		return truncateString(cell, maxWidth)
	}

	// Truncate and add "..."
	return truncateString(cell, maxWidth-3) + "..."
}

// truncateString truncates a string to the specified number of runes
func truncateString(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}

	return string(runes[:maxRunes])
}
