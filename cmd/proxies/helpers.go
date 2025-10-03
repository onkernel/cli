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
	// Strip color codes to measure visible characters only
	naturalWidths := make([]int, numCols)
	for colIdx := 0; colIdx < numCols; colIdx++ {
		maxWidth := 0
		for _, row := range data {
			if colIdx < len(row) {
				// Strip ANSI color codes to get visible character count
				visibleText := pterm.RemoveColorFromString(row[colIdx])
				cellWidth := utf8.RuneCountInString(visibleText)
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

// distributeColumnWidths calculates optimal width for each column using a two-pass strategy:
// Pass 1: ID and short columns get their full natural width
// Pass 2: Long columns share the remaining space
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

	// Define threshold for "short" columns (these get priority)
	const shortColumnThreshold = 15

	// Pass 1: Give ID (index 0) and short columns their full natural width
	remainingWidth := availableWidth
	longColumnIndices := []int{}

	for i := 0; i < numCols; i++ {
		if i == 0 || naturalWidths[i] <= shortColumnThreshold {
			// Short column or ID - give full natural width
			result[i] = naturalWidths[i]
			remainingWidth -= naturalWidths[i]
		} else {
			// Long column - defer to pass 2
			longColumnIndices = append(longColumnIndices, i)
		}
	}

	// Pass 2: Distribute remaining space among long columns
	if len(longColumnIndices) == 0 {
		return result
	}

	// Calculate how much long columns want
	totalLongNatural := 0
	totalLongMin := 0
	for _, idx := range longColumnIndices {
		totalLongNatural += naturalWidths[idx]
		totalLongMin += minWidths[idx]
	}

	if totalLongNatural <= remainingWidth {
		// Long columns fit naturally
		for _, idx := range longColumnIndices {
			result[idx] = naturalWidths[idx]
		}
		return result
	}

	if totalLongMin > remainingWidth {
		// Even minimums don't fit, distribute equally
		for _, idx := range longColumnIndices {
			result[idx] = remainingWidth / len(longColumnIndices)
			if result[idx] < 5 {
				result[idx] = 5 // absolute minimum
			}
		}
		return result
	}

	// Give long columns minimum, then distribute remainder proportionally
	extraSpace := remainingWidth - totalLongMin
	extraNeed := totalLongNatural - totalLongMin

	for _, idx := range longColumnIndices {
		result[idx] = minWidths[idx]
		if extraNeed > 0 {
			additionalNeed := naturalWidths[idx] - minWidths[idx]
			additionalGrant := (additionalNeed * extraSpace) / extraNeed
			result[idx] += additionalGrant
		}
	}

	return result
}

// truncateCell truncates a cell to maxWidth, adding "..." if truncated
// Handles ANSI color codes properly by measuring visible characters only
func truncateCell(cell string, maxWidth int) string {
	// Strip ANSI codes to measure visible width
	visibleText := pterm.RemoveColorFromString(cell)
	cellWidth := utf8.RuneCountInString(visibleText)

	if cellWidth <= maxWidth {
		return cell
	}

	// Cell needs truncation
	// If the cell has color codes, we need to be careful about truncation
	// For simplicity, strip colors, truncate, then return without color
	if maxWidth <= 3 {
		// Too narrow for "...", just truncate
		return truncateString(visibleText, maxWidth)
	}

	// Truncate and add "..."
	return truncateString(visibleText, maxWidth-3) + "..."
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
