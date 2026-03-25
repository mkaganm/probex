package report

import (
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// HTMLReporter generates human-readable HTML reports.
type HTMLReporter struct{}

// NewHTML creates a new HTMLReporter.
func NewHTML() *HTMLReporter { return &HTMLReporter{} }

// Format returns the reporter's format name.
func (r *HTMLReporter) Format() string { return "html" }

// htmlData is the view-model passed to the HTML template.
type htmlData struct {
	Summary     *models.RunSummary
	GeneratedAt string
	ScanDate    string
	DurationStr string
	BySeverity  []severityEntry
	ByCategory  []categoryEntry
	MaxSeverity int
	MaxCategory int
}

type severityEntry struct {
	Name    string
	Count   int
	Percent float64
}

type categoryEntry struct {
	Name    string
	Count   int
	Percent float64
}

// Generate writes an HTML report to the writer.
func (r *HTMLReporter) Generate(summary *models.RunSummary, w io.Writer) error {
	funcMap := template.FuncMap{
		"statusColor": func(s models.TestStatus) string {
			switch s {
			case models.StatusPassed:
				return "#00FF88"
			case models.StatusFailed:
				return "#FF4444"
			case models.StatusError:
				return "#FFD700"
			case models.StatusSkipped:
				return "#00D4FF"
			default:
				return "#888888"
			}
		},
		"durationSec": func(d time.Duration) string {
			return fmt.Sprintf("%.3fs", d.Seconds())
		},
		"severityColor": func(s models.Severity) string {
			switch s {
			case models.SeverityCritical:
				return "#FF4444"
			case models.SeverityHigh:
				return "#FF8844"
			case models.SeverityMedium:
				return "#FFD700"
			case models.SeverityLow:
				return "#00D4FF"
			case models.SeverityInfo:
				return "#888888"
			default:
				return "#888888"
			}
		},
		"add": func(a, b int) int { return a + b },
		"headerString": func(h map[string]string) string {
			if len(h) == 0 {
				return "(none)"
			}
			s := ""
			for k, v := range h {
				s += k + ": " + v + "\n"
			}
			return s
		},
		"assertionIcon": func(passed bool) string {
			if passed {
				return "PASS"
			}
			return "FAIL"
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	data := htmlData{
		Summary:     summary,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05 MST"),
		ScanDate:    summary.StartedAt.Format("2006-01-02 15:04:05 MST"),
		DurationStr: fmt.Sprintf("%.2fs", summary.Duration.Seconds()),
	}

	// Build severity breakdown.
	total := summary.TotalTests
	if total == 0 {
		total = 1 // avoid div-by-zero
	}
	severityOrder := []models.Severity{
		models.SeverityCritical, models.SeverityHigh, models.SeverityMedium,
		models.SeverityLow, models.SeverityInfo,
	}
	for _, sev := range severityOrder {
		count := summary.BySeverity[sev]
		if count > 0 {
			data.BySeverity = append(data.BySeverity, severityEntry{
				Name:    string(sev),
				Count:   count,
				Percent: float64(count) * 100.0 / float64(total),
			})
			if count > data.MaxSeverity {
				data.MaxSeverity = count
			}
		}
	}

	// Build category breakdown.
	categoryOrder := []models.TestCategory{
		models.CategoryHappyPath, models.CategoryEdgeCase, models.CategorySecurity,
		models.CategoryFuzz, models.CategoryRelation, models.CategoryConcurrency,
		models.CategoryPerformance,
	}
	for _, cat := range categoryOrder {
		count := summary.ByCategory[cat]
		if count > 0 {
			data.ByCategory = append(data.ByCategory, categoryEntry{
				Name:    string(cat),
				Count:   count,
				Percent: float64(count) * 100.0 / float64(summary.TotalTests),
			})
			if count > data.MaxCategory {
				data.MaxCategory = count
			}
		}
	}

	return tmpl.Execute(w, data)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>PROBEX Test Report</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, sans-serif;
    background: #0d1117; color: #c9d1d9; line-height: 1.6;
  }
  .container { max-width: 1200px; margin: 0 auto; padding: 20px; }

  /* Header */
  .header {
    background: linear-gradient(135deg, #161b22 0%, #0d1117 100%);
    border: 1px solid #30363d; border-radius: 12px;
    padding: 30px; margin-bottom: 24px; text-align: center;
  }
  .header h1 { font-size: 2rem; color: #00FF88; margin-bottom: 8px; }
  .header .subtitle { color: #8b949e; font-size: 0.95rem; }
  .header .meta { margin-top: 12px; color: #8b949e; font-size: 0.85rem; }

  /* Summary cards */
  .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 16px; margin-bottom: 24px; }
  .card {
    background: #161b22; border: 1px solid #30363d; border-radius: 10px;
    padding: 20px; text-align: center;
  }
  .card .value { font-size: 2rem; font-weight: 700; }
  .card .label { font-size: 0.85rem; color: #8b949e; margin-top: 4px; }
  .card-total .value { color: #c9d1d9; }
  .card-passed .value { color: #00FF88; }
  .card-failed .value { color: #FF4444; }
  .card-errors .value { color: #FFD700; }
  .card-skipped .value { color: #00D4FF; }
  .card-duration .value { color: #c9d1d9; font-size: 1.4rem; }

  /* Breakdown sections */
  .breakdown { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
  @media (max-width: 768px) { .breakdown { grid-template-columns: 1fr; } }
  .breakdown-panel {
    background: #161b22; border: 1px solid #30363d; border-radius: 10px; padding: 20px;
  }
  .breakdown-panel h3 { margin-bottom: 16px; color: #c9d1d9; font-size: 1rem; }
  .bar-row { display: flex; align-items: center; margin-bottom: 10px; }
  .bar-label { width: 100px; font-size: 0.85rem; color: #8b949e; text-transform: capitalize; }
  .bar-track { flex: 1; height: 22px; background: #21262d; border-radius: 4px; overflow: hidden; margin: 0 10px; }
  .bar-fill { height: 100%; border-radius: 4px; transition: width 0.3s; }
  .bar-count { width: 40px; font-size: 0.85rem; text-align: right; color: #8b949e; }

  /* Severity colors */
  .sev-critical { background: #FF4444; }
  .sev-high { background: #FF8844; }
  .sev-medium { background: #FFD700; }
  .sev-low { background: #00D4FF; }
  .sev-info { background: #888888; }

  /* Category colors */
  .cat-bar { background: #00FF88; }

  /* Results table */
  .results-section {
    background: #161b22; border: 1px solid #30363d; border-radius: 10px;
    padding: 20px; margin-bottom: 24px;
  }
  .results-section h3 { margin-bottom: 16px; }
  table { width: 100%; border-collapse: collapse; }
  thead th {
    text-align: left; padding: 10px 12px; border-bottom: 1px solid #30363d;
    font-size: 0.8rem; color: #8b949e; text-transform: uppercase; letter-spacing: 0.5px;
  }
  tbody tr { border-bottom: 1px solid #21262d; cursor: pointer; }
  tbody tr:hover { background: #1c2128; }
  td { padding: 10px 12px; font-size: 0.9rem; }
  .badge {
    display: inline-block; padding: 2px 10px; border-radius: 12px;
    font-size: 0.75rem; font-weight: 600; text-transform: uppercase;
  }
  .severity-badge {
    display: inline-block; padding: 2px 8px; border-radius: 10px;
    font-size: 0.7rem; font-weight: 600; text-transform: uppercase;
    background: #21262d;
  }

  /* Expandable details */
  .detail-row { display: none; }
  .detail-row.open { display: table-row; }
  .detail-cell { padding: 0 12px 16px 12px; }
  .detail-content {
    background: #0d1117; border: 1px solid #30363d; border-radius: 8px; padding: 16px;
  }
  .detail-content h4 { color: #00FF88; font-size: 0.85rem; margin: 12px 0 6px 0; }
  .detail-content h4:first-child { margin-top: 0; }
  .detail-content pre {
    background: #161b22; padding: 10px; border-radius: 6px; font-size: 0.8rem;
    overflow-x: auto; color: #8b949e; white-space: pre-wrap; word-break: break-all;
  }
  .assertion-list { list-style: none; padding: 0; }
  .assertion-list li { padding: 4px 0; font-size: 0.85rem; }
  .assertion-pass { color: #00FF88; }
  .assertion-fail { color: #FF4444; }

  /* Footer */
  .footer {
    text-align: center; padding: 20px; color: #484f58; font-size: 0.8rem;
  }
</style>
</head>
<body>
<div class="container">

  <div class="header">
    <h1>PROBEX</h1>
    <div class="subtitle">Zero-Test API Intelligence Engine &mdash; Test Report</div>
    <div class="meta">
      Scan date: {{.ScanDate}} &bull;
      Profile: {{.Summary.ProfileID}} &bull;
      Duration: {{.DurationStr}}
    </div>
  </div>

  <div class="cards">
    <div class="card card-total"><div class="value">{{.Summary.TotalTests}}</div><div class="label">Total Tests</div></div>
    <div class="card card-passed"><div class="value">{{.Summary.Passed}}</div><div class="label">Passed</div></div>
    <div class="card card-failed"><div class="value">{{.Summary.Failed}}</div><div class="label">Failed</div></div>
    <div class="card card-errors"><div class="value">{{.Summary.Errors}}</div><div class="label">Errors</div></div>
    <div class="card card-skipped"><div class="value">{{.Summary.Skipped}}</div><div class="label">Skipped</div></div>
    <div class="card card-duration"><div class="value">{{.DurationStr}}</div><div class="label">Duration</div></div>
  </div>

  <div class="breakdown">
    <div class="breakdown-panel">
      <h3>Severity Breakdown</h3>
      {{range .BySeverity}}
      <div class="bar-row">
        <span class="bar-label">{{.Name}}</span>
        <div class="bar-track"><div class="bar-fill sev-{{.Name}}" style="width: {{printf "%.1f" .Percent}}%"></div></div>
        <span class="bar-count">{{.Count}}</span>
      </div>
      {{end}}
      {{if not .BySeverity}}<p style="color:#484f58">No data</p>{{end}}
    </div>
    <div class="breakdown-panel">
      <h3>Category Breakdown</h3>
      {{range .ByCategory}}
      <div class="bar-row">
        <span class="bar-label">{{.Name}}</span>
        <div class="bar-track"><div class="bar-fill cat-bar" style="width: {{printf "%.1f" .Percent}}%"></div></div>
        <span class="bar-count">{{.Count}}</span>
      </div>
      {{end}}
      {{if not .ByCategory}}<p style="color:#484f58">No data</p>{{end}}
    </div>
  </div>

  <div class="results-section">
    <h3>Detailed Results</h3>
    <table>
      <thead>
        <tr>
          <th>#</th><th>Test Name</th><th>Status</th><th>Category</th><th>Severity</th><th>Duration</th>
        </tr>
      </thead>
      <tbody>
        {{range $i, $r := .Summary.Results}}
        <tr onclick="toggleDetail('d{{$i}}')">
          <td>{{add $i 1}}</td>
          <td>{{$r.TestName}}</td>
          <td><span class="badge" style="background:{{statusColor $r.Status}}; color:#0d1117;">{{$r.Status}}</span></td>
          <td>{{$r.Category}}</td>
          <td><span class="severity-badge" style="color:{{severityColor $r.Severity}}">{{$r.Severity}}</span></td>
          <td>{{durationSec $r.Duration}}</td>
        </tr>
        <tr class="detail-row" id="d{{$i}}">
          <td colspan="6" class="detail-cell">
            <div class="detail-content">
              <h4>Request</h4>
              <pre>{{$r.Request.Method}} {{$r.Request.URL}}
Headers: {{headerString $r.Request.Headers}}{{if $r.Request.Body}}
Body: {{$r.Request.Body}}{{end}}</pre>
              {{if $r.Response}}
              <h4>Response</h4>
              <pre>Status: {{$r.Response.StatusCode}}
Headers: {{headerString $r.Response.Headers}}{{if $r.Response.Body}}
Body: {{$r.Response.Body}}{{end}}</pre>
              {{end}}
              {{if $r.Error}}
              <h4>Error</h4>
              <pre>{{$r.Error}}</pre>
              {{end}}
              <h4>Assertions</h4>
              <ul class="assertion-list">
                {{range $r.Assertions}}
                <li class="{{if .Passed}}assertion-pass{{else}}assertion-fail{{end}}">
                  [{{assertionIcon .Passed}}] {{.Assertion.Type}} — {{.Assertion.Target}} {{.Assertion.Operator}} {{.Assertion.Expected}}
                  {{if .Message}} — {{.Message}}{{end}}
                </li>
                {{end}}
                {{if not $r.Assertions}}<li style="color:#484f58">No assertions</li>{{end}}
              </ul>
            </div>
          </td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>

  <div class="footer">
    Generated by PROBEX on {{.GeneratedAt}}
  </div>

</div>

<script>
function toggleDetail(id) {
  var el = document.getElementById(id);
  if (el) el.classList.toggle('open');
}
</script>
</body>
</html>
`
