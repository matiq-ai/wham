package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// errorWriter is a helper struct that wraps an io.Writer and tracks the first
// error that occurs during a sequence of writes. This allows for cleaner code
// by avoiding an `if err != nil` check after every single print statement.
type errorWriter struct {
	w   io.Writer
	err error
}

// Printf formats and writes to the underlying writer, but only if no error has
// occurred yet. If a new error occurs, it is stored.
func (ew *errorWriter) Printf(format string, a ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, a...)
}

// Println is a convenience wrapper around Printf for writing lines.
func (ew *errorWriter) Println(a ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, a...)
}

// TableRenderer helps build and render clean, kubectl-style tables.
type TableRenderer struct {
	ew        *errorWriter
	headers   []string
	rows      [][]string
	maxWidths []int
}

// NewTableRenderer creates a new table renderer.
func NewTableRenderer(w io.Writer, headers ...string) *TableRenderer {
	maxWidths := make([]int, len(headers))
	for i, h := range headers {
		maxWidths[i] = len(h)
	}
	return &TableRenderer{
		ew:        &errorWriter{w: w},
		headers:   headers,
		maxWidths: maxWidths,
	}
}

// AddRow adds a row of cells to the table. It automatically updates the maximum
// width for each column to ensure proper alignment during rendering.
func (tr *TableRenderer) AddRow(cells ...string) {
	tr.rows = append(tr.rows, cells)
	for i, cell := range cells {
		if len(cell) > tr.maxWidths[i] {
			tr.maxWidths[i] = len(cell)
		}
	}
}

// Render prints the complete, formatted table to the writer.
func (tr *TableRenderer) Render() error {
	if len(tr.headers) == 0 {
		return nil // Nothing to render
	}

	// Get terminal width to avoid ugly line wraps.
	// If not in a TTY (e.g., piping to a file), use a large default width.
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 120 // A reasonable default if we can't get the size.
	}

	// Calculate the width needed for all columns except the last one.
	// This includes the content and the "  " separators between columns.
	fixedWidth := 0
	numCols := len(tr.headers)
	if numCols > 1 {
		for i := 0; i < numCols-1; i++ {
			fixedWidth += tr.maxWidths[i] + 2 // width + separator
		}
	}

	// The last column gets the remaining space.
	lastColMaxWidth := termWidth - fixedWidth
	// Ensure the last column has at least some minimum width, and cap its max width.
	if numCols > 0 {
		if lastColMaxWidth < 10 {
			lastColMaxWidth = 10 // Give it at least a little space
		}
		if tr.maxWidths[numCols-1] > lastColMaxWidth {
			tr.maxWidths[numCols-1] = lastColMaxWidth
		}
	}

	// Build the format string for a row, e.g., "%-*s  %-*s  %-*s"
	var fmtParts []string
	for range tr.headers {
		fmtParts = append(fmtParts, "%-*s")
	}
	rowFmt := strings.Join(fmtParts, "  ")

	// Prepare arguments for the header. The args slice needs to be of type []any.
	// It will be interleaved: [width1, header1, width2, header2, ...]
	headerArgs := make([]any, 0, len(tr.headers)*2)
	for i, h := range tr.headers {
		headerArgs = append(headerArgs, tr.maxWidths[i], h)
	}
	tr.ew.Printf(rowFmt+"\n", headerArgs...)

	// Print each data row
	for _, row := range tr.rows {
		rowArgs := make([]any, 0, len(row)*2)
		for i, cell := range row {
			// For the last column, truncate if the cell content is wider than the allowed max width.
			if i == numCols-1 && len(cell) > tr.maxWidths[i] {
				if tr.maxWidths[i] > 3 {
					cell = cell[:tr.maxWidths[i]-3] + "..."
				} else {
					cell = cell[:tr.maxWidths[i]]
				}
			}
			rowArgs = append(rowArgs, tr.maxWidths[i], cell)
		}
		tr.ew.Printf(rowFmt+"\n", rowArgs...)
	}

	return tr.ew.err
}

// RenderData marshals the given data structure into the specified format (json or yaml)
// and writes it to the provided writer. It centralizes the logic for structured output.
func RenderData(w io.Writer, data any, format string) error {
	var output []byte
	var err error

	switch format {
	case "json":
		output, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data to JSON: %w", err)
		}
	case "yaml":
		output, err = yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data to YAML: %w", err)
		}
	default:
		// This function is only for structured formats. The caller should handle 'table'.
		return fmt.Errorf("unsupported structured output format: '%s'", format)
	}

	_, err = fmt.Fprintln(w, string(output))
	return err
}
