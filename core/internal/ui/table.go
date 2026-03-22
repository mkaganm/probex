package ui

import (
	"fmt"
	"strings"
)

// Table renders a simple ASCII table.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new Table with the given headers.
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{headers: headers, widths: widths}
}

// AddRow appends a row to the table.
func (t *Table) AddRow(cells ...string) {
	for i, c := range cells {
		if i < len(t.widths) && len(c) > t.widths[i] {
			t.widths[i] = len(c)
		}
	}
	t.rows = append(t.rows, cells)
}

// Render returns the table as a formatted string.
func (t *Table) Render() string {
	var b strings.Builder

	// Header
	t.renderRow(&b, t.headers)
	// Separator
	for i, w := range t.widths {
		if i > 0 {
			b.WriteString("─┼─")
		}
		b.WriteString(strings.Repeat("─", w))
	}
	b.WriteString("\n")
	// Rows
	for _, row := range t.rows {
		t.renderRow(&b, row)
	}
	return b.String()
}

func (t *Table) renderRow(b *strings.Builder, cells []string) {
	for i, w := range t.widths {
		if i > 0 {
			b.WriteString(" │ ")
		}
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		fmt.Fprintf(b, "%-*s", w, cell)
	}
	b.WriteString("\n")
}
