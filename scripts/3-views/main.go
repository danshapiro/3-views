package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

var labels = []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"}

const subagentInstruction = `<instructions>
You are one of several independent agents answering the same request.

Inspect the repository and available local context as needed.

Treat the repository as read-only. Use read-only commands and inspection tools.

When additional information or an action is needed from the parent agent, include an "Investigation requests for parent agent" section with exact commands, files, logs, hosts, credentials, or data needed.

Answer the user's query directly and completely.
</instructions>`

type modelsConfig map[string]string

type permEntry struct {
	Read             map[string]interface{} `json:"read"`
	Grep             interface{}            `json:"grep"`
	Glob             interface{}            `json:"glob"`
	Lsp              interface{}            `json:"lsp"`
	Edit             interface{}            `json:"edit"`
	Bash             map[string]string      `json:"bash"`
	ExternalDirectory interface{}           `json:"external_directory"`
}

type agentResult struct {
	Label      string
	Model      string
	DurationMs int64
	Completed  bool
	Content    string
	Err        error
	StderrPath string
	OutputPath string
}

type metadata struct {
	RunID           string            `json:"run_id"`
	Timestamp       string            `json:"timestamp"`
	AgentsRequested int               `json:"agents_requested"`
	TimeoutMinutes  int               `json:"timeout_minutes"`
	Labels          []string          `json:"labels"`
	QueryFile       string            `json:"query_file"`
	Results         map[string]result `json:"results"`
}

type result struct {
	Status     string `json:"status"`
	Model      string `json:"model"`
	DurationMs int64  `json:"duration_ms"`
	OutputFile string `json:"output_file,omitempty"`
	Error      string `json:"error,omitempty"`
}

func defaultPermConfig() permEntry {
	return permEntry{
		Read: map[string]interface{}{
			"*":             "allow",
			"*.env":         "deny",
			"*.env.*":       "deny",
			"*.env.example": "allow",
		},
		Grep:             "allow",
		Glob:             "allow",
		Lsp:              "allow",
		Edit:             "deny",
		Bash: map[string]string{
			"git status*": "allow",
			"git diff*":   "allow",
			"git log*":    "allow",
			"git show*":   "allow",
			"grep *":      "allow",
			"find *":      "allow",
			"ls *":        "allow",
			"cat *":       "allow",
			"*":           "deny",
		},
		ExternalDirectory: "allow",
	}
}

func findSkillRoot() string {
	if root := os.Getenv("3_VIEWS_ROOT"); root != "" {
		return root
	}
	
	// Use runtime.Caller to find the location of main.go, which works flawlessly with 'go run'
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		return filepath.Join(filepath.Dir(filename), "..", "..")
	}

	// Fallback to executable path
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 3; i++ { // check up to 3 levels up
		if _, err := os.Stat(filepath.Join(dir, "config", "models.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(filepath.Dir(exe), "..")
}

func loadModels(skillRoot string) (modelsConfig, error) {
	path := filepath.Join(skillRoot, "config", "models.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading models config %s: %w", path, err)
	}
	var mc modelsConfig
	if err := json.Unmarshal(data, &mc); err != nil {
		return nil, fmt.Errorf("parsing models config: %w", err)
	}
	return mc, nil
}

func loadQuery(queryFile, queryInline string) (string, string, error) {
	if queryFile != "" && queryInline != "" {
		return "", "", fmt.Errorf("--query and --query-file are mutually exclusive")
	}
	if queryFile != "" {
		data, err := os.ReadFile(queryFile)
		if err != nil {
			return "", "", fmt.Errorf("reading query file: %w", err)
		}
		return strings.TrimSpace(string(data)), queryFile, nil
	}
	return queryInline, "inline", nil
}

func runAgent(ctx context.Context, label, model, promptFile, cwd, runDir string, permJSON []byte, result *agentResult) {
	start := time.Now()
	result.Label = label
	result.Model = model

	stderrPath := filepath.Join(runDir, label+".stderr.log")
	stdoutPath := filepath.Join(runDir, label+".md")
	result.StderrPath = stderrPath
	result.OutputPath = stdoutPath

	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		result.Err = fmt.Errorf("creating stderr file: %w", err)
		return
	}
	defer stderrFile.Close()

	args := []string{"run", "--model", model, "--dir", cwd, "-f", promptFile, "--", "Follow the instructions in the attached prompt.txt file."}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "OPENCODE_PERMISSION="+string(permJSON))
	cmd.Stderr = stderrFile

	output, err := cmd.Output()
	result.DurationMs = time.Since(start).Milliseconds()

	if len(output) > 0 {
		result.Content = string(output)
		if wErr := os.WriteFile(stdoutPath, output, 0o644); wErr != nil {
			result.Err = fmt.Errorf("writing output file %s: %w", stdoutPath, wErr)
			return
		}
	}

	if err != nil {
		if ctx.Err() != nil {
			result.Completed = false
			return
		}
		result.Err = fmt.Errorf("agent %s failed: %w", label, err)
		return
	}

	result.Completed = true

	// output was already written and stored in result.Content if len > 0
}

func main() {
	queryFile := flag.String("query-file", "", "Path to file containing the query")
	queryInline := flag.String("query", "", "Inline query text")
	cwd := flag.String("cwd", "", "Working directory for subagents (required)")
	agentCount := flag.Int("agents", 3, "Number of agents to launch (1-6)")
	outDir := flag.String("out-dir", "", "Output directory (default: OS temp dir)")
	timeoutMin := flag.Int("timeout", 60, "Wall-clock timeout in minutes")
	flag.Parse()

	if *queryFile == "" && *queryInline == "" {
		fmt.Fprintln(os.Stderr, "Error: --query-file or --query required")
		os.Exit(1)
	}
	if *cwd == "" {
		fmt.Fprintln(os.Stderr, "Error: --cwd required")
		os.Exit(1)
	}
	if *agentCount < 1 {
		*agentCount = 1
	}
	if *agentCount > 6 {
		*agentCount = 6
	}

	if _, err := exec.LookPath("opencode"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: opencode binary not found in PATH")
		os.Exit(1)
	}

	skillRoot := findSkillRoot()
	models, err := loadModels(skillRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	activeLabels := labels[:*agentCount]
	for _, label := range activeLabels {
		if _, ok := models[label]; !ok {
			fmt.Fprintf(os.Stderr, "Error: no model configured for label %q\n", label)
			os.Exit(1)
		}
	}

	queryText, queryRef, err := loadQuery(*queryFile, *queryInline)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runDir, err := os.MkdirTemp(*outDir, "3-views-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating run directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(runDir, "query.txt"), []byte(queryText), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing query file: %v\n", err)
		os.Exit(1)
	}

	fullPrompt := fmt.Sprintf("<user_query>\n%s\n</user_query>\n\n%s", queryText, subagentInstruction)
	promptFile := filepath.Join(runDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte(fullPrompt), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing prompt file: %v\n", err)
		os.Exit(1)
	}

	permConfig := defaultPermConfig()
	permJSON, err := json.Marshal(permConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling permission config: %v\n", err)
		os.Exit(1)
	}

	timeout := time.Duration(*timeoutMin) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	runID := fmt.Sprintf("%d", time.Now().UnixMilli())

	fmt.Printf("===== 3-VIEWS RUN: %s =====\n", runID)
	fmt.Printf("Run directory: %s\n", runDir)
	fmt.Printf("Agents requested: %d\n", *agentCount)
	fmt.Printf("Timeout: %d minutes\n\n", *timeoutMin)

	results := make([]agentResult, *agentCount)
	var wg sync.WaitGroup

	for i, label := range activeLabels {
		wg.Add(1)
		go func(idx int, label, model string) {
			defer wg.Done()
			runAgent(ctx, label, model, promptFile, *cwd, runDir, permJSON, &results[idx])
		}(i, label, models[label])
	}

	wg.Wait()

	md := metadata{
		RunID:           runID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		AgentsRequested: *agentCount,
		TimeoutMinutes:  *timeoutMin,
		Labels:          activeLabels,
		QueryFile:       queryRef,
		Results:         make(map[string]result),
	}

	for _, r := range results {
		fmt.Printf("===== AGENT %s RESULT =====\n", strings.ToUpper(r.Label))
		fmt.Printf("Saved response: %s\n\n", r.OutputPath)

		if r.Completed {
			fmt.Println(r.Content)
			md.Results[r.Label] = result{Status: "completed", Model: r.Model, DurationMs: r.DurationMs, OutputFile: r.Label + ".md"}
		} else if r.Err != nil {
			fmt.Printf("STATUS: failed\nERROR: %v\nSee log: %s\n", r.Err, r.StderrPath)
			md.Results[r.Label] = result{Status: "failed", Model: r.Model, DurationMs: r.DurationMs, Error: r.Err.Error()}
		} else {
			fmt.Printf("STATUS: timed out\nERROR: Reached %d minute run timeout before this agent completed.\nSee log: %s\n", *timeoutMin, r.StderrPath)
			md.Results[r.Label] = result{Status: "timed_out", Model: r.Model, DurationMs: r.DurationMs, Error: "timeout"}
		}
		fmt.Println()
	}

	fmt.Println("===== 3-VIEWS END =====")

	mdData, err := json.MarshalIndent(md, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling metadata: %v\n", err)
	} else if err := os.WriteFile(filepath.Join(runDir, "metadata.json"), mdData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing metadata.json: %v\n", err)
	}
}