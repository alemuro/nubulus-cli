package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// PrintStructured formats data according to format (table, json, yaml).
func PrintStructured(w io.Writer, format Format, data any, printTableFn func(io.Writer) error) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case FormatYAML:
		return yaml.NewEncoder(w).Encode(data)
	case FormatTable, "":
		if printTableFn != nil {
			return printTableFn(w)
		}
		return yaml.NewEncoder(w).Encode(data)
	default:
		return fmt.Errorf("format de sortida desconegut: %s (utilitzeu 'table', 'json' o 'yaml')", format)
	}
}

// NewTableWriter returns a standard tabwriter for CLI tables.
func NewTableWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 8, 3, ' ', 0)
}

// Success prints a green success message.
func Success(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "\033[32m✔\033[0m "+format+"\n", a...)
}

// Info prints a blue informational message.
func Info(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "\033[34mℹ\033[0m "+format+"\n", a...)
}

// Warn prints a yellow warning message.
func Warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "\033[33m⚠\033[0m "+format+"\n", a...)
}

// Error prints a red error message.
func Error(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "\033[31m✖ Error:\033[0m "+format+"\n", a...)
}
