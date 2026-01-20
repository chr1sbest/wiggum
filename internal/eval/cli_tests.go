package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLITestResult represents a single test result
type CLITestResult struct {
	Name    string
	Passed  bool
	Message string
}

// CLITestRunner runs tests for CLI tool projects
type CLITestRunner struct {
	ProjectDir  string
	FixturesDir string
	Binary      string
	Results     []CLITestResult
	Passed      int
	Failed      int
}

// NewCLITestRunner creates a new CLI test runner
func NewCLITestRunner(projectDir, fixturesDir string) *CLITestRunner {
	return &CLITestRunner{
		ProjectDir:  projectDir,
		FixturesDir: fixturesDir,
		Results:     []CLITestResult{},
	}
}

// RunTest executes a command and checks if output contains expected string
func (r *CLITestRunner) RunTest(name, cmd, expected string) {
	c := exec.Command("bash", "-c", cmd)
	c.Dir = r.ProjectDir
	output, _ := c.CombinedOutput()
	outputStr := string(output)

	passed := strings.Contains(outputStr, expected)
	if passed {
		fmt.Printf("  %s... ✅ PASS\n", name)
		r.Passed++
	} else {
		fmt.Printf("  %s... ❌ FAIL\n", name)
		fmt.Printf("    Expected to find: %s\n", expected)
		if len(outputStr) > 200 {
			outputStr = outputStr[:200]
		}
		fmt.Printf("    Got: %s\n", outputStr)
		r.Failed++
	}
	r.Results = append(r.Results, CLITestResult{Name: name, Passed: passed})
}

// RunTestExitCode checks if command exits with expected code
func (r *CLITestRunner) RunTestExitCode(name, cmd string, expectedCode int) {
	c := exec.Command("bash", "-c", cmd)
	c.Dir = r.ProjectDir
	err := c.Run()

	actualCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			actualCode = exitErr.ExitCode()
		} else {
			actualCode = 1
		}
	}

	passed := actualCode == expectedCode
	if passed {
		fmt.Printf("  %s... ✅ PASS\n", name)
		r.Passed++
	} else {
		fmt.Printf("  %s... ❌ FAIL (exit code %d, expected %d)\n", name, actualCode, expectedCode)
		r.Failed++
	}
	r.Results = append(r.Results, CLITestResult{Name: name, Passed: passed})
}

// CommandExists checks if a command/subcommand exists
func (r *CLITestRunner) CommandExists(cmd string) bool {
	c := exec.Command("bash", "-c", cmd)
	c.Dir = r.ProjectDir
	output, _ := c.CombinedOutput()
	return len(output) > 0
}

// FailMissing records a failure for a missing/unimplemented feature
func (r *CLITestRunner) FailMissing(name, feature string) {
	fmt.Printf("  %s... ❌ FAIL (not implemented)\n", name)
	r.Failed++
	r.Results = append(r.Results, CLITestResult{Name: name, Passed: false, Message: feature + " not implemented"})
}

// GetTotal returns total number of tests
func (r *CLITestRunner) GetTotal() int {
	return r.Passed + r.Failed
}

// SaveResults saves results to JSON file
func (r *CLITestRunner) SaveResults() error {
	results := map[string]interface{}{
		"suite":  "logagg",
		"passed": r.Passed,
		"failed": r.Failed,
		"total":  r.GetTotal(),
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(r.ProjectDir, ".eval_results.json"), data, 0644)
}

// RunLogaggTests runs the logagg CLI test suite
func RunLogaggTests(projectDir, fixturesDir string) (*TestResult, error) {
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║            Log Aggregator Eval Suite                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Printf("Project: %s\n", projectDir)
	fmt.Println("")

	r := NewCLITestRunner(projectDir, fixturesDir)

	// Find and build the binary
	fmt.Println("🔨 Build Tests")

	buildPath := ""
	if fileExists(filepath.Join(projectDir, "cmd", "logagg", "main.go")) {
		buildPath = "./cmd/logagg"
	} else if fileExists(filepath.Join(projectDir, "main.go")) {
		buildPath = "."
	}

	if buildPath == "" {
		fmt.Println("  ❌ FAIL: Cannot find main.go")
		r.Failed++
	} else {
		r.RunTestExitCode("go build succeeds", fmt.Sprintf("go build -o logagg %s", buildPath), 0)
	}

	// Find binary
	binaryPath := ""
	if fileExists(filepath.Join(projectDir, "logagg")) {
		binaryPath = "./logagg"
	} else if fileExists(filepath.Join(projectDir, "bin", "logagg")) {
		binaryPath = "./bin/logagg"
	}

	if binaryPath == "" {
		fmt.Println("  ❌ FAIL: Binary not found after build")
		r.Failed++
		return &TestResult{Passed: r.Passed, Failed: r.Failed, Total: r.GetTotal()}, nil
	}
	r.Binary = binaryPath

	// Help tests
	fmt.Println("")
	fmt.Println("📖 Help Tests")
	r.RunTest("help command shows usage", fmt.Sprintf("%s --help", r.Binary), "Usage")
	r.RunTest("parse subcommand exists", fmt.Sprintf("%s parse --help 2>&1 || %s --help", r.Binary, r.Binary), "parse")

	// Define fixture paths
	jsonLog := filepath.Join(fixturesDir, "json.log")
	apacheLog := filepath.Join(fixturesDir, "apache.log")
	syslogLog := filepath.Join(fixturesDir, "syslog.log")

	// JSON parse tests
	fmt.Println("")
	fmt.Println("📄 JSON Parse Tests")
	r.RunTest("parse JSON logs", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "Application started")
	r.RunTest("parse JSON shows error level", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "error")
	r.RunTest("parse JSON shows timestamp", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "2024")
	r.RunTest("parse JSON shows source field", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "api")
	r.RunTest("parse JSON shows warn level", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "warn")
	r.RunTest("parse JSON shows debug level", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "debug")
	r.RunTest("parse JSON shows message content", fmt.Sprintf("%s parse %s --format json 2>&1 || %s parse %s 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "Connection refused")

	// Apache parse tests
	fmt.Println("")
	fmt.Println("🌐 Apache Parse Tests")
	r.RunTest("parse Apache logs", fmt.Sprintf("%s parse %s --format apache 2>&1 || %s parse %s 2>&1", r.Binary, apacheLog, r.Binary, apacheLog), "GET")
	r.RunTest("parse Apache shows IP", fmt.Sprintf("%s parse %s --format apache 2>&1 || %s parse %s 2>&1", r.Binary, apacheLog, r.Binary, apacheLog), "127.0.0.1")
	r.RunTest("parse Apache shows POST method", fmt.Sprintf("%s parse %s --format apache 2>&1 || %s parse %s 2>&1", r.Binary, apacheLog, r.Binary, apacheLog), "POST")
	r.RunTest("parse Apache shows path", fmt.Sprintf("%s parse %s --format apache 2>&1 || %s parse %s 2>&1", r.Binary, apacheLog, r.Binary, apacheLog), "/api/login")

	// Syslog parse tests
	fmt.Println("")
	fmt.Println("📋 Syslog Parse Tests")
	r.RunTest("parse Syslog logs", fmt.Sprintf("%s parse %s --format syslog 2>&1 || %s parse %s 2>&1", r.Binary, syslogLog, r.Binary, syslogLog), "sshd")
	r.RunTest("parse Syslog shows hostname", fmt.Sprintf("%s parse %s --format syslog 2>&1 || %s parse %s 2>&1", r.Binary, syslogLog, r.Binary, syslogLog), "myserver")
	r.RunTest("parse Syslog shows kernel", fmt.Sprintf("%s parse %s --format syslog 2>&1 || %s parse %s 2>&1", r.Binary, syslogLog, r.Binary, syslogLog), "kernel")

	// Auto-detection tests
	fmt.Println("")
	fmt.Println("🔮 Format Auto-Detection Tests")
	r.RunTest("auto-detect JSON format", fmt.Sprintf("%s parse %s 2>&1", r.Binary, jsonLog), "Application started")
	r.RunTest("auto-detect Apache format", fmt.Sprintf("%s parse %s 2>&1", r.Binary, apacheLog), "GET")

	// Filter tests
	fmt.Println("")
	fmt.Println("🔍 Filter Tests")
	r.RunTest("filter by error level", fmt.Sprintf("%s filter %s --level error 2>&1", r.Binary, jsonLog), "error")
	r.RunTest("filter excludes non-matching", fmt.Sprintf("%s filter %s --level error 2>&1 | grep -c 'info' || echo '0'", r.Binary, jsonLog), "0")
	r.RunTest("filter by warn level", fmt.Sprintf("%s filter %s --level warn 2>&1", r.Binary, jsonLog), "warn")
	r.RunTest("filter by info level", fmt.Sprintf("%s filter %s --level info 2>&1", r.Binary, jsonLog), "info")
	r.RunTest("filter case insensitive ERROR", fmt.Sprintf("%s filter %s --level ERROR 2>&1 || %s filter %s --level error 2>&1", r.Binary, jsonLog, r.Binary, jsonLog), "error")
	// Test regex matching
	r.RunTest("filter by regex match", fmt.Sprintf("%s filter %s --match 'Connection' 2>&1", r.Binary, jsonLog), "Connection")
	r.RunTest("filter by regex match timeout", fmt.Sprintf("%s filter %s --match 'Timeout' 2>&1", r.Binary, jsonLog), "Timeout")

	// Stats tests
	fmt.Println("")
	fmt.Println("📊 Stats Tests")
	r.RunTest("stats shows total count", fmt.Sprintf("%s stats %s 2>&1", r.Binary, jsonLog), "10")
	r.RunTest("stats shows level breakdown", fmt.Sprintf("%s stats %s 2>&1", r.Binary, jsonLog), "error")
	r.RunTest("stats shows info count", fmt.Sprintf("%s stats %s 2>&1", r.Binary, jsonLog), "info")
	r.RunTest("stats shows warn count", fmt.Sprintf("%s stats %s 2>&1", r.Binary, jsonLog), "warn")
	// Test group-by
	r.RunTest("stats group by level", fmt.Sprintf("%s stats %s --group-by level 2>&1", r.Binary, jsonLog), "info")
	r.RunTest("stats group by source", fmt.Sprintf("%s stats %s --group-by source 2>&1", r.Binary, jsonLog), "api")

	// Error handling tests
	fmt.Println("")
	fmt.Println("⚠️  Error Handling Tests")
	r.RunTest("error on missing file", fmt.Sprintf("%s parse /nonexistent/file.log 2>&1", r.Binary), "Error")
	r.RunTest("error message is helpful", fmt.Sprintf("%s parse /nonexistent/file.log 2>&1", r.Binary), "no such file")

	// Output format tests
	fmt.Println("")
	fmt.Println("📤 Output Format Tests")
	r.RunTest("output as JSON", fmt.Sprintf("%s parse %s --output json 2>&1 | head -5", r.Binary, jsonLog), "{")
	r.RunTest("output as CSV has header", fmt.Sprintf("%s parse %s --output csv 2>&1 | head -1", r.Binary, jsonLog), "timestamp")

	// Merge tests
	fmt.Println("")
	fmt.Println("🔀 Merge Tests")
	r.RunTest("merge two files", fmt.Sprintf("%s merge %s %s 2>&1 | head -5", r.Binary, jsonLog, apacheLog), "2024")

	// Query tests
	fmt.Println("")
	fmt.Println("🔎 Query Tests")
	r.RunTest("query SELECT works", fmt.Sprintf("%s query %s \"SELECT level FROM logs\" 2>&1", r.Binary, jsonLog), "info")
	r.RunTest("query with WHERE", fmt.Sprintf("%s query %s \"SELECT * FROM logs WHERE level='error'\" 2>&1", r.Binary, jsonLog), "error")

	// Save results
	if err := r.SaveResults(); err != nil {
		fmt.Printf("WARNING: failed to save results: %v\n", err)
	}

	fmt.Println("")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("  Results: %d passed, %d failed out of %d\n", r.Passed, r.Failed, r.GetTotal())
	fmt.Println("═══════════════════════════════════════════════════════════════")

	return &TestResult{
		Passed: r.Passed,
		Failed: r.Failed,
		Total:  r.GetTotal(),
	}, nil
}
