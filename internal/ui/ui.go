package ui

import (
	"fmt"
	"io"
	"os"
)

const (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[0;33m"
	cyan   = "\033[0;36m"
	reset  = "\033[0m"
)

// Out is the writer used for log/ok/warn/err output. Defaults to stderr so
// stdout stays clean for machine-readable output (e.g. commit messages).
var Out io.Writer = os.Stderr

func colorize(c, s string) string {
	if !useColor() {
		return s
	}
	return c + s + reset
}

func useColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := Out.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Log prints an informational message.
func Log(format string, a ...any) {
	fmt.Fprintf(Out, "%s %s\n", colorize(cyan, "▸"), fmt.Sprintf(format, a...))
}

// OK prints a success message.
func OK(format string, a ...any) {
	fmt.Fprintf(Out, "%s %s\n", colorize(green, "✔"), fmt.Sprintf(format, a...))
}

// Warn prints a warning message.
func Warn(format string, a ...any) {
	fmt.Fprintf(Out, "%s %s\n", colorize(yellow, "⚠"), fmt.Sprintf(format, a...))
}

// Errf prints an error and exits with code 1.
func Errf(format string, a ...any) {
	fmt.Fprintf(Out, "%s %s\n", colorize(red, "✖"), fmt.Sprintf(format, a...))
	os.Exit(1)
}

// Banner prints a horizontal divider line.
func Banner() {
	fmt.Fprintln(Out, colorize(yellow, "────────────────────────────"))
}

// Highlight returns text in green.
func Highlight(s string) string { return colorize(green, s) }

// Prompt returns text in cyan, suitable for interactive prompts.
func Prompt(s string) string { return colorize(cyan, s) }

// Color identifies one of the palette colors used by Log/OK/Warn/Errf.
type Color int

const (
	ColorGreen Color = iota
	ColorRed
	ColorYellow
	ColorCyan
)

// Colorize wraps s in the given palette color, honoring NO_COLOR and
// non-terminal output the same way Log/OK/Warn/Errf do. Used for glyphs and
// other status indicators that don't fit the Log/OK/Warn/Errf shapes.
func Colorize(c Color, s string) string {
	switch c {
	case ColorGreen:
		return colorize(green, s)
	case ColorRed:
		return colorize(red, s)
	case ColorYellow:
		return colorize(yellow, s)
	case ColorCyan:
		return colorize(cyan, s)
	default:
		return s
	}
}
