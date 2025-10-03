package cmd

import (
	"strings"
	"unicode/utf8"

	"github.com/pterm/pterm"
)

// printTableNoPad renders a table similar to pterm.DefaultTable, but it avoids
// adding trailing padding spaces after the last column and does not add blank
// padded lines to match multi-line cells in other columns. The last column may
// contain multi-line content which will be printed as-is on following lines.
// It also intelligently truncates columns to prevent line wrapping.
func printTableNoPad(data pterm.TableData, hasHeader bool) {
	if len(data) == 0 {
		return
	}

	// Get terminal width and truncate data to fit
	termWidth := pterm.GetTerminalWidth()
	if termWidth <= 0 {
		termWidth = 80 // fallback
	}
	data = truncateTableData(data, termWidth)

	// Determine number of columns from the first row
	numCols := len(data[0])
	if numCols == 0 {
		return
	}

	// Pre-compute max width per column for all but the last column
	maxColWidths := make([]int, numCols)
	for _, row := range data {
		for colIdx := 0; colIdx < numCols && colIdx < len(row); colIdx++ {
			if colIdx == numCols-1 {
				continue
			}
			for _, line := range strings.Split(row[colIdx], "\n") {
				if w := utf8.RuneCountInString(line); w > maxColWidths[colIdx] {
					maxColWidths[colIdx] = w
				}
			}
		}
	}

	var b strings.Builder
	sep := pterm.DefaultTable.Separator
	sepStyled := pterm.ThemeDefault.TableSeparatorStyle.Sprint(sep)

	renderRow := func(row []string, styleHeader bool) {
		// Build first-line-only for non-last columns; last column is full string
		firstLineParts := make([]string, 0, numCols)
		for colIdx := 0; colIdx < numCols; colIdx++ {
			var cell string
			if colIdx < len(row) {
				cell = row[colIdx]
			}

			if colIdx < numCols-1 {
				// Only the first line for non-last columns
				lines := strings.Split(cell, "\n")
				first := ""
				if len(lines) > 0 {
					first = lines[0]
				}
				padCount := maxColWidths[colIdx] - utf8.RuneCountInString(first)
				if padCount < 0 {
					padCount = 0
				}
				firstLineParts = append(firstLineParts, first+strings.Repeat(" ", padCount))
			} else {
				// Last column: render the first line now; remaining lines after
				lines := strings.Split(cell, "\n")
				if len(lines) > 0 {
					firstLineParts = append(firstLineParts, lines[0])
				} else {
					firstLineParts = append(firstLineParts, "")
				}
			}
		}

		line := strings.Join(firstLineParts[:numCols-1], sepStyled)
		if numCols > 1 {
			if line != "" {
				line += sepStyled
			}
			line += firstLineParts[numCols-1]
		}

		if styleHeader {
			b.WriteString(pterm.ThemeDefault.TableHeaderStyle.Sprint(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")

		// Print remaining lines from the last column without alignment padding
		if numCols > 0 {
			var lastCell string
			if len(row) >= numCols {
				lastCell = row[numCols-1]
			}
			lines := strings.Split(lastCell, "\n")
			if len(lines) > 1 {
				rest := strings.Join(lines[1:], "\n")
				if rest != "" {
					b.WriteString(rest)
					b.WriteString("\n")
				}
			}
		}
	}

	for idx, row := range data {
		renderRow(row, hasHeader && idx == 0)
	}

	pterm.Print(b.String())
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
// The first column (typically ID) is always given its full natural width and never truncated
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

	// Priority: Always give the first column its full natural width
	if numCols > 0 {
		result[0] = naturalWidths[0]
		availableWidth -= naturalWidths[0]
	}

	// Now distribute remaining width among other columns (excluding first)
	if numCols <= 1 {
		return result
	}

	// Calculate needs for non-first columns
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
