package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/probex/probex/internal/storage"
)

// Dashboard serves a web UI for viewing PROBEX results.
type Dashboard struct {
	store *storage.Store
}

// New creates a new Dashboard.
func New(store *storage.Store) *Dashboard {
	return &Dashboard{store: store}
}

// RegisterHandlers registers dashboard HTTP handlers on the given mux.
func (d *Dashboard) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard", d.handleDashboard)
	mux.HandleFunc("GET /dashboard/api/summary", d.handleAPISummary)
	mux.HandleFunc("GET /dashboard/api/runs", d.handleAPIRuns)
}

func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, nil)
}

func (d *Dashboard) handleAPISummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	summary, err := d.store.LoadResults()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "no results"})
		return
	}

	// Build a lightweight summary.
	resp := map[string]any{
		"total":    summary.TotalTests,
		"passed":   summary.Passed,
		"failed":   summary.Failed,
		"errors":   summary.Errors,
		"skipped":  summary.Skipped,
		"duration": fmt.Sprintf("%.2fs", summary.Duration.Seconds()),
		"profile":  summary.ProfileID,
	}

	// Top failures.
	var failures []map[string]any
	for _, r := range summary.Results {
		if r.Status == "failed" || r.Status == "error" {
			failures = append(failures, map[string]any{
				"name":     r.TestName,
				"status":   string(r.Status),
				"severity": string(r.Severity),
				"category": string(r.Category),
				"error":    r.Error,
			})
		}
	}
	resp["failures"] = failures

	// By severity.
	bySev := make(map[string]int)
	for k, v := range summary.BySeverity {
		bySev[string(k)] = v
	}
	resp["by_severity"] = bySev

	// By category.
	byCat := make(map[string]int)
	for k, v := range summary.ByCategory {
		byCat[string(k)] = v
	}
	resp["by_category"] = byCat

	json.NewEncoder(w).Encode(resp)
}

func (d *Dashboard) handleAPIRuns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	runs, err := d.store.ListRuns()
	if err != nil {
		json.NewEncoder(w).Encode([]any{})
		return
	}

	type runEntry struct {
		Name      string `json:"name"`
		Timestamp string `json:"timestamp"`
	}
	var entries []runEntry
	for _, run := range runs {
		entries = append(entries, runEntry{
			Name:      run.Name,
			Timestamp: run.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}
	json.NewEncoder(w).Encode(entries)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>PROBEX Dashboard</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #0d1117; color: #c9d1d9; min-height: 100vh;
  }
  .nav {
    background: #161b22; border-bottom: 1px solid #30363d;
    padding: 16px 24px; display: flex; align-items: center; gap: 16px;
  }
  .nav h1 { color: #00FF88; font-size: 1.3rem; }
  .nav .version { color: #484f58; font-size: 0.8rem; }
  .container { max-width: 1400px; margin: 0 auto; padding: 24px; }

  .stats { display: grid; grid-template-columns: repeat(6, 1fr); gap: 16px; margin-bottom: 24px; }
  @media (max-width: 900px) { .stats { grid-template-columns: repeat(3, 1fr); } }
  .stat {
    background: #161b22; border: 1px solid #30363d; border-radius: 10px;
    padding: 20px; text-align: center;
  }
  .stat .val { font-size: 2.2rem; font-weight: 700; }
  .stat .lbl { font-size: 0.8rem; color: #8b949e; margin-top: 4px; }
  .stat.total .val { color: #c9d1d9; }
  .stat.pass .val { color: #00FF88; }
  .stat.fail .val { color: #FF4444; }
  .stat.err .val { color: #FFD700; }
  .stat.skip .val { color: #00D4FF; }
  .stat.dur .val { color: #c9d1d9; font-size: 1.5rem; }

  .panels { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
  @media (max-width: 768px) { .panels { grid-template-columns: 1fr; } }
  .panel {
    background: #161b22; border: 1px solid #30363d; border-radius: 10px; padding: 20px;
  }
  .panel h3 { margin-bottom: 16px; font-size: 1rem; }

  .chart-bar { display: flex; align-items: center; margin-bottom: 8px; }
  .chart-bar .label { width: 90px; font-size: 0.8rem; color: #8b949e; text-transform: capitalize; }
  .chart-bar .track { flex: 1; height: 24px; background: #21262d; border-radius: 4px; overflow: hidden; margin: 0 8px; }
  .chart-bar .fill { height: 100%; border-radius: 4px; transition: width 0.5s ease; min-width: 2px; }
  .chart-bar .count { width: 40px; text-align: right; font-size: 0.85rem; color: #8b949e; }

  .fill-critical { background: #FF4444; }
  .fill-high { background: #FF8844; }
  .fill-medium { background: #FFD700; }
  .fill-low { background: #00D4FF; }
  .fill-info { background: #888; }
  .fill-cat { background: #00FF88; }

  .failures {
    background: #161b22; border: 1px solid #30363d; border-radius: 10px; padding: 20px;
  }
  .failures h3 { margin-bottom: 16px; }
  .failure-item {
    border-bottom: 1px solid #21262d; padding: 12px 0;
  }
  .failure-item:last-child { border-bottom: none; }
  .failure-name { font-weight: 600; margin-bottom: 4px; }
  .failure-meta { font-size: 0.8rem; color: #8b949e; }
  .failure-error { font-size: 0.8rem; color: #FF4444; margin-top: 4px; font-family: monospace; }
  .badge {
    display: inline-block; padding: 2px 8px; border-radius: 10px;
    font-size: 0.7rem; font-weight: 600; text-transform: uppercase; margin-right: 6px;
  }
  .badge-failed { background: #FF444433; color: #FF4444; }
  .badge-error { background: #FFD70033; color: #FFD700; }
  .badge-critical { background: #FF444433; color: #FF4444; }
  .badge-high { background: #FF884433; color: #FF8844; }
  .badge-medium { background: #FFD70033; color: #FFD700; }

  .empty { text-align: center; color: #484f58; padding: 40px; }
  .empty h2 { margin-bottom: 8px; }

  .refresh-btn {
    background: #21262d; border: 1px solid #30363d; color: #c9d1d9;
    padding: 8px 16px; border-radius: 6px; cursor: pointer; font-size: 0.85rem;
  }
  .refresh-btn:hover { background: #30363d; }

  .runs { margin-top: 16px; }
  .run-item { font-size: 0.85rem; padding: 4px 0; color: #8b949e; }
</style>
</head>
<body>
<div class="nav">
  <h1>PROBEX</h1>
  <span class="version">Dashboard v1.0</span>
  <div style="flex:1"></div>
  <button class="refresh-btn" onclick="loadData()">Refresh</button>
</div>

<div class="container">
  <div id="content">
    <div class="empty">
      <h2>Loading...</h2>
    </div>
  </div>
</div>

<script>
async function loadData() {
  try {
    const [summaryRes, runsRes] = await Promise.all([
      fetch('/dashboard/api/summary'),
      fetch('/dashboard/api/runs')
    ]);
    const summary = await summaryRes.json();
    const runs = await runsRes.json();

    if (summary.error) {
      document.getElementById('content').innerHTML =
        '<div class="empty"><h2>No Results Yet</h2><p>Run <code>probex scan</code> and <code>probex run</code> first.</p></div>';
      return;
    }

    renderDashboard(summary, runs);
  } catch (e) {
    document.getElementById('content').innerHTML =
      '<div class="empty"><h2>Connection Error</h2><p>' + e.message + '</p></div>';
  }
}

function renderDashboard(s, runs) {
  const total = s.total || 1;
  const bySev = s.by_severity || {};
  const byCat = s.by_category || {};
  const failures = s.failures || [];
  const maxSev = Math.max(...Object.values(bySev), 1);
  const maxCat = Math.max(...Object.values(byCat), 1);

  let html = '';

  // Stats
  html += '<div class="stats">';
  html += '<div class="stat total"><div class="val">' + s.total + '</div><div class="lbl">Total</div></div>';
  html += '<div class="stat pass"><div class="val">' + s.passed + '</div><div class="lbl">Passed</div></div>';
  html += '<div class="stat fail"><div class="val">' + s.failed + '</div><div class="lbl">Failed</div></div>';
  html += '<div class="stat err"><div class="val">' + s.errors + '</div><div class="lbl">Errors</div></div>';
  html += '<div class="stat skip"><div class="val">' + s.skipped + '</div><div class="lbl">Skipped</div></div>';
  html += '<div class="stat dur"><div class="val">' + s.duration + '</div><div class="lbl">Duration</div></div>';
  html += '</div>';

  // Charts
  html += '<div class="panels">';

  // Severity
  html += '<div class="panel"><h3>By Severity</h3>';
  ['critical','high','medium','low','info'].forEach(sev => {
    const count = bySev[sev] || 0;
    if (count > 0) {
      const pct = (count / maxSev * 100).toFixed(1);
      html += '<div class="chart-bar"><span class="label">' + sev + '</span>';
      html += '<div class="track"><div class="fill fill-' + sev + '" style="width:' + pct + '%"></div></div>';
      html += '<span class="count">' + count + '</span></div>';
    }
  });
  html += '</div>';

  // Category
  html += '<div class="panel"><h3>By Category</h3>';
  Object.entries(byCat).sort((a,b) => b[1]-a[1]).forEach(([cat, count]) => {
    const pct = (count / maxCat * 100).toFixed(1);
    html += '<div class="chart-bar"><span class="label">' + cat.replace('_',' ') + '</span>';
    html += '<div class="track"><div class="fill fill-cat" style="width:' + pct + '%"></div></div>';
    html += '<span class="count">' + count + '</span></div>';
  });
  html += '</div>';
  html += '</div>';

  // Failures
  if (failures.length > 0) {
    html += '<div class="failures"><h3>Failures (' + failures.length + ')</h3>';
    failures.slice(0, 20).forEach(f => {
      html += '<div class="failure-item">';
      html += '<div class="failure-name">' + escapeHtml(f.name) + '</div>';
      html += '<div class="failure-meta">';
      html += '<span class="badge badge-' + f.status + '">' + f.status + '</span>';
      html += '<span class="badge badge-' + f.severity + '">' + f.severity + '</span>';
      html += f.category;
      html += '</div>';
      if (f.error) html += '<div class="failure-error">' + escapeHtml(f.error) + '</div>';
      html += '</div>';
    });
    html += '</div>';
  }

  // Runs
  if (runs && runs.length > 0) {
    html += '<div class="panel" style="margin-top:16px"><h3>Recent Runs</h3><div class="runs">';
    runs.slice(0, 10).forEach(r => {
      html += '<div class="run-item">' + r.timestamp + ' — ' + r.name + '</div>';
    });
    html += '</div></div>';
  }

  document.getElementById('content').innerHTML = html;
}

function escapeHtml(s) {
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

loadData();
setInterval(loadData, 30000);
</script>
</body>
</html>`
