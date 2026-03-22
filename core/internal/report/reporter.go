package report

import (
	"fmt"
	"io"

	"github.com/probex/probex/internal/models"
)

// Reporter is the common interface for all report generators.
type Reporter interface {
	Generate(summary *models.RunSummary, w io.Writer) error
	Format() string
}

// NewReporter returns a Reporter for the given format string.
// Supported formats: "json", "junit", "html".
func NewReporter(format string) (Reporter, error) {
	switch format {
	case "json":
		return NewJSON(), nil
	case "junit":
		return NewJUnit(), nil
	case "html":
		return NewHTML(), nil
	default:
		return nil, fmt.Errorf("unsupported report format: %q (supported: json, junit, html)", format)
	}
}
