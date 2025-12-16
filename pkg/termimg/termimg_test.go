package termimg

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalType_String(t *testing.T) {
	tests := []struct {
		term     TerminalType
		expected string
	}{
		{TerminalUnknown, "Unknown"},
		{TerminaliTerm2, "iTerm2"},
		{TerminalKitty, "Kitty"},
		{TerminalGhostty, "Ghostty"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.term.String())
	}
}

func TestDetectTerminal_iTerm2(t *testing.T) {
	// Save and restore env vars
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Setenv("TERM_PROGRAM", "iTerm.app")
	os.Unsetenv("KITTY_WINDOW_ID")

	assert.Equal(t, TerminaliTerm2, DetectTerminal())
	assert.True(t, IsSupported())
}

func TestDetectTerminal_Kitty(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Unsetenv("TERM_PROGRAM")
	os.Setenv("KITTY_WINDOW_ID", "12345")

	assert.Equal(t, TerminalKitty, DetectTerminal())
	assert.True(t, IsSupported())
}

func TestDetectTerminal_Ghostty(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Setenv("TERM_PROGRAM", "ghostty")
	os.Unsetenv("KITTY_WINDOW_ID")

	assert.Equal(t, TerminalGhostty, DetectTerminal())
	assert.True(t, IsSupported())
}

func TestDetectTerminal_Unknown(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Unsetenv("TERM_PROGRAM")
	os.Unsetenv("KITTY_WINDOW_ID")

	assert.Equal(t, TerminalUnknown, DetectTerminal())
	assert.False(t, IsSupported())
}

func TestDisplayImage_UnsupportedTerminal(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Unsetenv("TERM_PROGRAM")
	os.Unsetenv("KITTY_WINDOW_ID")

	var buf bytes.Buffer
	err := DisplayImage(&buf, []byte("fake image data"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminal does not support inline images")
}

func TestDisplayImage_iTerm2(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Setenv("TERM_PROGRAM", "iTerm.app")
	os.Unsetenv("KITTY_WINDOW_ID")

	var buf bytes.Buffer
	imgData := []byte("test png data")
	err := DisplayImage(&buf, imgData)

	require.NoError(t, err)
	output := buf.String()
	// Should contain iTerm2 escape sequence prefix
	assert.Contains(t, output, "\033]1337;File=inline=1;width=auto;height=auto;preserveAspectRatio=1:")
	// Should end with bell character
	assert.True(t, output[len(output)-1] == '\a')
}

func TestDisplayImage_Kitty(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Unsetenv("TERM_PROGRAM")
	os.Setenv("KITTY_WINDOW_ID", "12345")

	var buf bytes.Buffer
	imgData := []byte("test png data")
	err := DisplayImage(&buf, imgData)

	require.NoError(t, err)
	output := buf.String()
	// Should contain Kitty escape sequence prefix
	assert.Contains(t, output, "\033_Ga=T,f=100")
}

func TestDisplayImage_Ghostty(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Setenv("TERM_PROGRAM", "ghostty")
	os.Unsetenv("KITTY_WINDOW_ID")

	var buf bytes.Buffer
	imgData := []byte("test png data")
	err := DisplayImage(&buf, imgData)

	require.NoError(t, err)
	output := buf.String()
	// Ghostty uses Kitty protocol, should contain Kitty escape sequence prefix
	assert.Contains(t, output, "\033_Ga=T,f=100")
}

func TestDisplayImage_Kitty_LargeImage(t *testing.T) {
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyID := os.Getenv("KITTY_WINDOW_ID")
	defer func() {
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyID)
	}()

	os.Unsetenv("TERM_PROGRAM")
	os.Setenv("KITTY_WINDOW_ID", "12345")

	var buf bytes.Buffer
	// Create data that will result in > 4096 bytes when base64 encoded
	// (4096 * 3/4 = 3072 raw bytes, so use more)
	imgData := make([]byte, 5000)
	for i := range imgData {
		imgData[i] = byte(i % 256)
	}
	err := DisplayImage(&buf, imgData)

	require.NoError(t, err)
	output := buf.String()
	// Should have multiple chunks indicated by m=1 (more) followed by m=0 (last)
	assert.Contains(t, output, "m=1")
	assert.Contains(t, output, "m=0")
}
