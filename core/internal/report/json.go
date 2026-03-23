package report

import (
	"encoding/json"
	"io"

	"github.com/mkaganm/probex/internal/models"
)

// JSONReporter generates JSON reports from test results.
type JSONReporter struct{}

// NewJSON creates a new JSONReporter.
func NewJSON() *JSONReporter { return &JSONReporter{} }

// Format returns the reporter's format name.
func (r *JSONReporter) Format() string { return "json" }

// Generate writes a JSON report to the writer.
func (r *JSONReporter) Generate(summary *models.RunSummary, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}
