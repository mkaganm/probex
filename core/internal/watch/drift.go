package watch

import (
	"fmt"

	"github.com/probex/probex/internal/models"
)

// DriftDetector detects schema changes in API responses.
type DriftDetector struct{}

// NewDriftDetector creates a new DriftDetector.
func NewDriftDetector() *DriftDetector {
	return &DriftDetector{}
}

// Drift represents a detected schema change.
type Drift struct {
	EndpointID string          `json:"endpoint_id"`
	Field      string          `json:"field"`
	Change     string          `json:"change"` // added, removed, type_changed, format_changed
	OldValue   string          `json:"old_value,omitempty"`
	NewValue   string          `json:"new_value,omitempty"`
	Severity   models.Severity `json:"severity"`
}

// Compare recursively compares a known schema against an actual (observed) schema,
// returning any detected drifts.
func (d *DriftDetector) Compare(endpointID string, known, actual *models.Schema) []Drift {
	if known == nil || actual == nil {
		return nil
	}
	return d.compareSchemas(endpointID, "", known, actual)
}

func (d *DriftDetector) compareSchemas(endpointID, prefix string, known, actual *models.Schema) []Drift {
	var drifts []Drift

	// Type change
	if known.Type != actual.Type && known.Type != "" && actual.Type != "" {
		drifts = append(drifts, Drift{
			EndpointID: endpointID,
			Field:      fieldPath(prefix, ""),
			Change:     "type_changed",
			OldValue:   known.Type,
			NewValue:   actual.Type,
			Severity:   models.SeverityHigh,
		})
		return drifts // No point comparing deeper if type changed
	}

	// Format change
	if known.Format != actual.Format && known.Format != "" {
		drifts = append(drifts, Drift{
			EndpointID: endpointID,
			Field:      fieldPath(prefix, ""),
			Change:     "format_changed",
			OldValue:   known.Format,
			NewValue:   actual.Format,
			Severity:   models.SeverityMedium,
		})
	}

	// Compare object properties
	if known.Type == "object" && known.Properties != nil {
		// Check for removed fields
		for name := range known.Properties {
			if actual.Properties == nil || actual.Properties[name] == nil {
				drifts = append(drifts, Drift{
					EndpointID: endpointID,
					Field:      fieldPath(prefix, name),
					Change:     "removed",
					OldValue:   known.Properties[name].Type,
					Severity:   models.SeverityHigh,
				})
			}
		}

		// Check for added or changed fields
		if actual.Properties != nil {
			for name, actualProp := range actual.Properties {
				knownProp := known.Properties[name]
				if knownProp == nil {
					drifts = append(drifts, Drift{
						EndpointID: endpointID,
						Field:      fieldPath(prefix, name),
						Change:     "added",
						NewValue:   actualProp.Type,
						Severity:   models.SeverityLow,
					})
				} else {
					// Recurse into matching fields
					subDrifts := d.compareSchemas(endpointID, fieldPath(prefix, name), knownProp, actualProp)
					drifts = append(drifts, subDrifts...)
				}
			}
		}
	}

	// Compare array items
	if known.Type == "array" && known.Items != nil && actual.Items != nil {
		subDrifts := d.compareSchemas(endpointID, prefix+"[]", known.Items, actual.Items)
		drifts = append(drifts, subDrifts...)
	}

	return drifts
}

func fieldPath(prefix, name string) string {
	if prefix == "" {
		if name == "" {
			return "(root)"
		}
		return name
	}
	if name == "" {
		return prefix
	}
	return fmt.Sprintf("%s.%s", prefix, name)
}
