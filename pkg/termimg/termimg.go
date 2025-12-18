// Package termimg provides utilities for displaying images inline in terminal emulators.
// It supports iTerm2 and Kitty graphics protocols.
package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// TerminalType represents the type of terminal emulator.
type TerminalType int

const (
	TerminalUnknown TerminalType = iota
	TerminaliTerm2
	TerminalKitty
	TerminalGhostty
)

func (t TerminalType) String() string {
	switch t {
	case TerminaliTerm2:
		return "iTerm2"
	case TerminalKitty:
		return "Kitty"
	case TerminalGhostty:
		return "Ghostty"
	default:
		return "Unknown"
	}
}

// DetectTerminal returns the type of terminal emulator based on environment variables.
func DetectTerminal() TerminalType {
	termProgram := os.Getenv("TERM_PROGRAM")
	// Check for iTerm2
	if termProgram == "iTerm.app" {
		return TerminaliTerm2
	}
	// Check for Ghostty (uses Kitty graphics protocol)
	if termProgram == "ghostty" {
		return TerminalGhostty
	}
	// Check for Kitty
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return TerminalKitty
	}
	return TerminalUnknown
}

// IsSupported returns true if the current terminal supports inline image display.
func IsSupported() bool {
	return DetectTerminal() != TerminalUnknown
}

// getTerminalSize returns the terminal width and height in columns and rows.
// Returns default values if the size cannot be determined.
func getTerminalSize() (cols, rows int) {
	// Default to reasonable values if we can't detect
	cols, rows = 80, 24

	// Try stdout first, then stdin
	for _, fd := range []int{int(os.Stdout.Fd()), int(os.Stdin.Fd())} {
		if term.IsTerminal(fd) {
			if w, h, err := term.GetSize(fd); err == nil {
				return w, h
			}
		}
	}
	return cols, rows
}

// DisplayImage writes escape sequences to display the given image data inline.
// The image data should be raw PNG/JPEG bytes.
func DisplayImage(w io.Writer, img []byte) error {
	term := DetectTerminal()
	switch term {
	case TerminaliTerm2:
		return displayiTerm2(w, img)
	case TerminalKitty, TerminalGhostty:
		// Ghostty uses the Kitty graphics protocol
		return displayKitty(w, img)
	default:
		return fmt.Errorf("terminal does not support inline images (detected: %s). Try using iTerm2, Kitty, or Ghostty, or use --to to save to a file", term)
	}
}

// ANSI escape sequences for cursor and screen control
const (
	cursorHide    = "\033[?25l"
	cursorShow    = "\033[?25h"
	cursorHome    = "\033[H"
	clearToEnd    = "\033[J"
	clearScreen   = "\033[2J"
	saveCursor    = "\033[s"
	restoreCursor = "\033[u"

	// Synchronized output mode (supported by Kitty, Ghostty, and others)
	// Buffers all output and renders atomically to prevent flicker
	syncStart = "\033[?2026h"
	syncEnd   = "\033[?2026l"
)

// HideCursor hides the terminal cursor.
func HideCursor(w io.Writer) {
	fmt.Fprint(w, cursorHide)
}

// ShowCursor shows the terminal cursor.
func ShowCursor(w io.Writer) {
	fmt.Fprint(w, cursorShow)
}

// ClearScreen clears the entire screen and moves cursor to home position.
func ClearScreen(w io.Writer) {
	fmt.Fprint(w, cursorHome+clearScreen)
}

// ClearAndDisplayImage clears the screen and displays an image at the top.
// This is used for animation/live view to replace the previous frame.
func ClearAndDisplayImage(w io.Writer, img []byte) error {
	term := DetectTerminal()
	switch term {
	case TerminaliTerm2:
		// For iTerm2: move cursor home, clear screen, then draw
		fmt.Fprint(w, cursorHome+clearToEnd)
		return displayiTerm2(w, img)
	case TerminalKitty, TerminalGhostty:
		// For Kitty: delete previous image by ID, then draw new one with same ID
		return displayKittyFrame(w, img)
	default:
		return fmt.Errorf("terminal does not support inline images (detected: %s). Try using iTerm2, Kitty, or Ghostty", term)
	}
}

// liveViewImageID is the placement ID used for live view frames
const liveViewImageID = 1

// displayKittyFrame displays an image for live view, replacing any previous frame.
// Uses synchronized output mode to prevent flicker - all drawing is buffered
// and rendered atomically.
func displayKittyFrame(w io.Writer, img []byte) error {
	encoded := base64.StdEncoding.EncodeToString(img)
	cols, _ := getTerminalSize()

	// Buffer all output to write in one go
	var buf bytes.Buffer

	// Start synchronized output mode - terminal buffers all changes
	buf.WriteString(syncStart)

	// Delete the previous frame with this ID (ignore if none exists)
	// a=d means delete, d=i means delete by image ID, i=ID specifies the ID
	// q=2 quiet mode - suppress response
	fmt.Fprintf(&buf, "\033_Ga=d,d=i,q=2,i=%d\033\\", liveViewImageID)

	// Move cursor to home position for consistent placement
	buf.WriteString(cursorHome)

	const chunkSize = 4096

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]

		more := 1
		if end >= len(encoded) {
			more = 0
		}

		if i == 0 {
			// First chunk: include all parameters
			// a=T transmit and display, f=100 PNG, i=ID placement ID, c=cols width
			// q=2 quiet mode - suppress response to avoid printing "_Gi=1;OK" garbage
			fmt.Fprintf(&buf, "\033_Ga=T,q=2,f=100,i=%d,c=%d,m=%d;%s\033\\", liveViewImageID, cols, more, chunk)
		} else {
			fmt.Fprintf(&buf, "\033_Gm=%d;%s\033\\", more, chunk)
		}
	}

	// End synchronized output mode - terminal renders everything atomically
	buf.WriteString(syncEnd)

	// Write everything in one go
	_, err := w.Write(buf.Bytes())
	return err
}

// CleanupLiveView cleans up after a live view session.
// It shows the cursor and optionally clears the last frame.
func CleanupLiveView(w io.Writer, clearImage bool) {
	term := DetectTerminal()

	// For Kitty/Ghostty, delete the live view image
	if clearImage && (term == TerminalKitty || term == TerminalGhostty) {
		fmt.Fprintf(w, "\033_Ga=d,d=i,q=2,i=%d\033\\", liveViewImageID)
	}

	// Show cursor again
	ShowCursor(w)

	// Print newline to ensure prompt appears on clean line
	fmt.Fprintln(w)
}

// displayiTerm2 renders an image using iTerm2's inline images protocol.
// Protocol: ESC ] 1337 ; File = [args] : base64data BEL
// https://iterm2.com/documentation-images.html
func displayiTerm2(w io.Writer, img []byte) error {
	encoded := base64.StdEncoding.EncodeToString(img)
	// inline=1 displays the image inline
	// width=100% fills terminal width, height=auto preserves aspect ratio
	// preserveAspectRatio=1 maintains aspect ratio
	_, err := fmt.Fprintf(w, "\033]1337;File=inline=1;width=100%%;height=auto;preserveAspectRatio=1:%s\a", encoded)
	return err
}

// displayKitty renders an image using Kitty's graphics protocol.
// Protocol uses chunked transmission for large images.
// https://sw.kovidgoyal.net/kitty/graphics-protocol/
func displayKitty(w io.Writer, img []byte) error {
	encoded := base64.StdEncoding.EncodeToString(img)

	// Get terminal width to scale image appropriately
	cols, _ := getTerminalSize()

	// Kitty requires chunked transmission for data over 4096 bytes
	const chunkSize = 4096

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]

		// m=1 means more chunks coming, m=0 means last chunk
		// a=T means transmit and display
		// f=100 means PNG format (also works for JPEG)
		// c=cols means display width in terminal columns
		// q=2 quiet mode - suppress response to avoid printing garbage
		if i == 0 {
			// First chunk includes all the parameters
			more := 1
			if end >= len(encoded) {
				more = 0
			}
			_, err := fmt.Fprintf(w, "\033_Ga=T,q=2,f=100,c=%d,m=%d;%s\033\\", cols, more, chunk)
			if err != nil {
				return err
			}
		} else {
			// Subsequent chunks only need the 'm' parameter
			more := 1
			if end >= len(encoded) {
				more = 0
			}
			_, err := fmt.Fprintf(w, "\033_Gm=%d;%s\033\\", more, chunk)
			if err != nil {
				return err
			}
		}
	}

	// Print a newline after the image so subsequent output appears below
	_, err := fmt.Fprintln(w)
	return err
}
