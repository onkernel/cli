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
func printTableNoPad(data pterm.TableData, hasHeader bool) {
	if len(data) == 0 {
		return
	}

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
