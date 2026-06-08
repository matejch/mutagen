package report

import (
	"bufio"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matej/mutagen/pkg/mutator"
)

// sourceFile holds source code with mutation annotations for the HTML report.
type sourceFile struct {
	Path     string
	RelPath  string
	Lines    []sourceLine
	Total    int
	Killed   int
	Survived int
	KillRate float64
}

type sourceLine struct {
	Number    int
	Content   string
	Mutations []lineMutation
	Status    string // "killed", "survived", "mixed", ""
}

type lineMutation struct {
	Description string
	Status      string
}

// WriteHTMLFile writes an HTML report to a file.
func WriteHTMLFile(path string, r Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	// Group results by file
	files := buildSourceFiles(r)

	data := struct {
		Summary Summary
		Files   []sourceFile
	}{
		Summary: r.Summary,
		Files:   files,
	}

	return htmlTemplate.Execute(w, data)
}

func buildSourceFiles(r Report) []sourceFile {
	byFile := make(map[string][]mutator.Result)
	for _, res := range r.Results {
		byFile[res.Mutation.File] = append(byFile[res.Mutation.File], res)
	}

	var files []sourceFile
	for filePath, results := range byFile {
		files = append(files, buildOneSourceFile(filePath, results))
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Survived != files[j].Survived {
			return files[i].Survived > files[j].Survived
		}
		return files[i].RelPath < files[j].RelPath
	})

	return files
}

func buildOneSourceFile(filePath string, results []mutator.Result) sourceFile {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return sourceFile{Path: filePath, RelPath: filepath.Base(filePath)}
	}

	rawLines := strings.Split(string(content), "\n")
	lines := make([]sourceLine, len(rawLines))
	for i, l := range rawLines {
		lines[i] = sourceLine{Number: i + 1, Content: l}
	}

	total, killed, survived := mapMutationsToLines(lines, results)
	setLineStatuses(lines)

	relPath := filePath
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, filePath); err == nil {
			relPath = rel
		}
	}

	var killRate float64
	if tested := killed + survived; tested > 0 {
		killRate = float64(killed) / float64(tested) * 100
	}

	return sourceFile{
		Path: filePath, RelPath: relPath, Lines: lines,
		Total: total, Killed: killed, Survived: survived, KillRate: killRate,
	}
}

func mapMutationsToLines(lines []sourceLine, results []mutator.Result) (total, killed, survived int) {
	for _, res := range results {
		lineIdx := res.Mutation.Line - 1
		if lineIdx < 0 || lineIdx >= len(lines) {
			continue
		}
		total++
		lines[lineIdx].Mutations = append(lines[lineIdx].Mutations, lineMutation{
			Description: res.Mutation.Description,
			Status:      res.Status.String(),
		})
		switch res.Status {
		case mutator.StatusKilled:
			killed++
		case mutator.StatusSurvived:
			survived++
		}
	}
	return
}

func setLineStatuses(lines []sourceLine) {
	for i := range lines {
		if len(lines[i].Mutations) == 0 {
			continue
		}
		var hasKilled, hasSurvived bool
		for _, m := range lines[i].Mutations {
			switch m.Status {
			case "killed":
				hasKilled = true
			case "survived":
				hasSurvived = true
			}
		}
		switch {
		case hasSurvived && hasKilled:
			lines[i].Status = "mixed"
		case hasSurvived:
			lines[i].Status = "survived"
		case hasKilled:
			lines[i].Status = "killed"
		}
	}
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"pct": func(f float64) string { return fmt.Sprintf("%.1f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Mutation Testing Report</title>
<style>
  :root {
    --bg: #0d1117; --fg: #c9d1d9; --border: #30363d;
    --killed: #238636; --survived: #da3633; --mixed: #d29922;
    --line-bg: #161b22; --hover: #1f242b;
    --code-bg: #0d1117;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
         background: var(--bg); color: var(--fg); line-height: 1.5; }

  .container { max-width: 1200px; margin: 0 auto; padding: 24px; }

  /* Summary */
  .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
             gap: 12px; margin-bottom: 32px; }
  .stat { background: var(--line-bg); border: 1px solid var(--border); border-radius: 8px;
          padding: 16px; text-align: center; }
  .stat-value { font-size: 2em; font-weight: 700; }
  .stat-label { font-size: 0.85em; color: #8b949e; }
  .stat-killed .stat-value { color: var(--killed); }
  .stat-survived .stat-value { color: var(--survived); }

  /* Progress bar */
  .kill-bar { height: 8px; border-radius: 4px; background: var(--survived);
              overflow: hidden; margin: 8px 0 24px; }
  .kill-bar-fill { height: 100%; background: var(--killed); border-radius: 4px; }

  /* File list */
  .file { margin-bottom: 24px; border: 1px solid var(--border); border-radius: 8px;
          overflow: hidden; }
  .file-header { background: var(--line-bg); padding: 12px 16px; display: flex;
                 justify-content: space-between; align-items: center; cursor: pointer;
                 border-bottom: 1px solid var(--border); }
  .file-header:hover { background: var(--hover); }
  .file-path { font-family: "SFMono-Regular", Consolas, monospace; font-size: 0.9em;
               font-weight: 600; }
  .file-stats { font-size: 0.85em; color: #8b949e; }
  .file-stats .survived-count { color: var(--survived); font-weight: 700; }

  /* Source code */
  .source { display: none; overflow-x: auto; }
  .file.open .source { display: block; }
  table { width: 100%; border-collapse: collapse; font-family: "SFMono-Regular", Consolas, monospace;
          font-size: 0.82em; }
  tr { border: none; }
  tr:hover { background: var(--hover); }
  td { padding: 0 12px; white-space: pre; vertical-align: top; }
  .line-num { color: #484f58; text-align: right; width: 1%; user-select: none;
              padding-right: 16px; border-right: 1px solid var(--border); }
  .line-code { padding-left: 16px; }

  tr.survived { background: rgba(218, 54, 51, 0.1); }
  tr.survived .line-num { color: var(--survived); }
  tr.killed { background: rgba(35, 134, 54, 0.05); }
  tr.mixed { background: rgba(210, 153, 34, 0.1); }

  .mutation-badge { display: inline-block; font-size: 0.75em; padding: 1px 6px;
                    border-radius: 3px; margin-left: 8px; vertical-align: middle; }
  .mutation-badge.killed { background: var(--killed); color: white; }
  .mutation-badge.survived { background: var(--survived); color: white; }
  .mutation-badge.build_error { background: #484f58; color: white; }
  .mutation-badge.timeout { background: var(--mixed); color: black; }

  /* Collapse toggle */
  .toggle { font-size: 0.8em; color: #8b949e; }
  .file.open .toggle::before { content: "▼ "; }
  .file:not(.open) .toggle::before { content: "▶ "; }

  h1 { font-size: 1.5em; margin-bottom: 4px; }
  .subtitle { color: #8b949e; margin-bottom: 24px; }
</style>
</head>
<body>
<div class="container">
  <h1>Mutation Testing Report</h1>
  <p class="subtitle">{{.Summary.Total}} mutations · {{pct .Summary.KillRate}}% killed</p>

  <div class="summary">
    <div class="stat stat-killed">
      <div class="stat-value">{{.Summary.Killed}}</div>
      <div class="stat-label">Killed</div>
    </div>
    <div class="stat stat-survived">
      <div class="stat-value">{{.Summary.Survived}}</div>
      <div class="stat-label">Survived</div>
    </div>
    <div class="stat">
      <div class="stat-value">{{.Summary.BuildError}}</div>
      <div class="stat-label">Build Errors</div>
    </div>
    <div class="stat">
      <div class="stat-value">{{.Summary.Timeout}}</div>
      <div class="stat-label">Timeouts</div>
    </div>
    <div class="stat">
      <div class="stat-value">{{pct .Summary.KillRate}}%</div>
      <div class="stat-label">Kill Rate</div>
    </div>
    <div class="stat">
      <div class="stat-value">{{pct .Summary.Duration}}s</div>
      <div class="stat-label">Duration</div>
    </div>
  </div>

  <div class="kill-bar"><div class="kill-bar-fill" style="width:{{pct .Summary.KillRate}}%"></div></div>

  {{range .Files}}
  <div class="file{{if .Survived}} open{{end}}">
    <div class="file-header" onclick="this.parentElement.classList.toggle('open')">
      <span>
        <span class="toggle"></span>
        <span class="file-path">{{.RelPath}}</span>
      </span>
      <span class="file-stats">
        {{.Total}} mutations ·
        {{.Killed}} killed ·
        {{if .Survived}}<span class="survived-count">{{.Survived}} survived</span>{{else}}0 survived{{end}}
        · {{pct .KillRate}}%
      </span>
    </div>
    <div class="source">
      <table>
        {{range .Lines}}
        <tr{{if .Status}} class="{{.Status}}"{{end}}>
          <td class="line-num">{{.Number}}</td>
          <td class="line-code">{{.Content}}{{range .Mutations}}<span class="mutation-badge {{.Status}}">{{.Description}}</span>{{end}}</td>
        </tr>
        {{end}}
      </table>
    </div>
  </div>
  {{end}}
</div>
</body>
</html>
`))
