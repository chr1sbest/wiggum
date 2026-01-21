package eval

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunWorkflowTests runs the workflow engine CLI test suite
func RunWorkflowTests(projectDir, fixturesDir string) (*TestResult, error) {
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║            Workflow Engine Eval Suite                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Printf("Project: %s\n", projectDir)
	fmt.Printf("Fixtures: %s\n", fixturesDir)
	fmt.Println("")

	r := NewCLITestRunner(projectDir, fixturesDir)

	// Clean up any retry test state from previous runs
	os.Remove("/tmp/workflow_retry_test")

	// ========== BUILD TESTS ==========
	fmt.Println("🔨 Build Tests")

	buildPath := findWorkflowBuildPath(projectDir)
	if buildPath == "" {
		fmt.Println("  ❌ FAIL: Cannot find main.go for workflow binary")
		r.Failed++
		r.Results = append(r.Results, CLITestResult{Name: "go build succeeds", Passed: false, Message: "main.go not found"})
	} else {
		r.RunTestExitCode("go build succeeds", fmt.Sprintf("go build -o workflow %s", buildPath), 0)
	}

	// Find binary
	binaryPath := findWorkflowBinary(projectDir)
	if binaryPath == "" {
		fmt.Println("  ❌ FAIL: Binary not found after build")
		r.Failed++
		return &TestResult{Passed: r.Passed, Failed: r.Failed, Total: r.GetTotal()}, nil
	}
	r.Binary = binaryPath

	// ========== HELP TESTS ==========
	fmt.Println("")
	fmt.Println("📖 Help & CLI Tests")
	r.RunTest("help shows usage", fmt.Sprintf("%s --help 2>&1 || %s help 2>&1 || %s 2>&1", r.Binary, r.Binary, r.Binary), "Usage")
	r.RunTest("has run subcommand", fmt.Sprintf("%s --help 2>&1 || %s 2>&1", r.Binary, r.Binary), "run")
	r.RunTest("has validate subcommand", fmt.Sprintf("%s --help 2>&1 || %s 2>&1", r.Binary, r.Binary), "validate")

	// ========== BASIC EXECUTION TESTS ==========
	fmt.Println("")
	fmt.Println("▶️  Basic Execution Tests")

	simpleYaml := filepath.Join(fixturesDir, "simple.yaml")
	r.RunTest("run simple workflow", fmt.Sprintf("%s run %s 2>&1", r.Binary, simpleYaml), "Hello")
	r.RunTestExitCode("simple workflow exits 0", fmt.Sprintf("%s run %s", r.Binary, simpleYaml), 0)

	seqYaml := filepath.Join(fixturesDir, "sequential.yaml")
	r.RunTest("run sequential workflow", fmt.Sprintf("%s run %s 2>&1", r.Binary, seqYaml), "Step 1")
	r.RunTest("sequential runs step 2", fmt.Sprintf("%s run %s 2>&1", r.Binary, seqYaml), "Step 2")
	r.RunTest("sequential runs step 3", fmt.Sprintf("%s run %s 2>&1", r.Binary, seqYaml), "Step 3")

	// ========== DEPENDENCY ORDER TESTS ==========
	fmt.Println("")
	fmt.Println("🔗 Dependency Order Tests")

	// Test that steps with dependencies run after their dependencies
	r.RunTestOutputOrder("dependency order correct", fmt.Sprintf("%s run %s 2>&1", r.Binary, seqYaml),
		[]string{"Step 1", "Step 2", "Step 3"})

	parallelYaml := filepath.Join(fixturesDir, "parallel.yaml")
	r.RunTest("parallel workflow completes", fmt.Sprintf("%s run %s 2>&1", r.Binary, parallelYaml), "All tasks done")
	r.RunTestOutputOrder("setup runs before tasks", fmt.Sprintf("%s run %s 2>&1", r.Binary, parallelYaml),
		[]string{"Setup complete", "All tasks done"})

	// ========== VARIABLE TESTS ==========
	fmt.Println("")
	fmt.Println("📝 Variable Tests")

	varsYaml := filepath.Join(fixturesDir, "variables.yaml")
	r.RunTest("interpolates vars.greeting", fmt.Sprintf("%s run %s 2>&1", r.Binary, varsYaml), "greet")
	r.RunTestExitCode("interpolates vars.target", fmt.Sprintf("%s run %s", r.Binary, varsYaml), 0)
	r.RunTest("interpolates vars.version", fmt.Sprintf("%s run %s 2>&1", r.Binary, varsYaml), "version")

	// Test --var override
	r.RunTestExitCode("--var overrides variable", fmt.Sprintf("%s run %s --var greeting=Goodbye 2>&1 || %s run %s -var greeting=Goodbye 2>&1", r.Binary, varsYaml, r.Binary, varsYaml), 0)

	// ========== ENVIRONMENT VARIABLE TESTS ==========
	fmt.Println("")
	fmt.Println("🌍 Environment Variable Tests")

	envYaml := filepath.Join(fixturesDir, "env_vars.yaml")
	r.RunTest("step env vars set", fmt.Sprintf("%s run %s 2>&1", r.Binary, envYaml), "with-env")
	r.RunTestExitCode("multiple env vars", fmt.Sprintf("%s run %s", r.Binary, envYaml), 0)

	// ========== CONDITION TESTS ==========
	fmt.Println("")
	fmt.Println("❓ Condition Tests")

	condYaml := filepath.Join(fixturesDir, "conditions.yaml")
	r.RunTestNotContains("condition true runs step", fmt.Sprintf("%s run %s 2>&1", r.Binary, condYaml), "deploy.*skipped")
	r.RunTestNotContains("condition false skips step", fmt.Sprintf("%s run %s 2>&1", r.Binary, condYaml), "This should be skipped")
	r.RunTest("condition interpolates vars", fmt.Sprintf("%s run %s 2>&1", r.Binary, condYaml), "deploy")

	// ========== FAILURE HANDLING TESTS ==========
	fmt.Println("")
	fmt.Println("💥 Failure Handling Tests")

	failYaml := filepath.Join(fixturesDir, "failure.yaml")
	r.RunTestExitCode("workflow with failure exits non-zero", fmt.Sprintf("%s run %s", r.Binary, failYaml), 1)
	r.RunTest("step after failure skipped", fmt.Sprintf("%s run %s 2>&1", r.Binary, failYaml), "skipped")
	r.RunTest("always() runs despite failure", fmt.Sprintf("%s run %s 2>&1", r.Binary, failYaml), "always-run")

	// ========== CONTINUE ON ERROR TESTS ==========
	fmt.Println("")
	fmt.Println("🔄 Continue On Error Tests")

	continueYaml := filepath.Join(fixturesDir, "continue_on_error.yaml")
	r.RunTestExitCode("continue_on_error workflow succeeds", fmt.Sprintf("%s run %s", r.Binary, continueYaml), 0)
	r.RunTestNotContains("next step runs after continue_on_error", fmt.Sprintf("%s run %s 2>&1", r.Binary, continueYaml), "skipped")

	// ========== FAILURE CONDITION TESTS ==========
	fmt.Println("")
	fmt.Println("🚨 Failure Condition Function Tests")

	failCondYaml := filepath.Join(fixturesDir, "failure_condition.yaml")
	r.RunTest("failure() condition runs on failure", fmt.Sprintf("%s run %s 2>&1", r.Binary, failCondYaml), "on-failure")
	r.RunTestNotContains("success() skips on failure", fmt.Sprintf("%s run %s 2>&1", r.Binary, failCondYaml), "This should be skipped")

	// ========== OUTPUT TESTS ==========
	fmt.Println("")
	fmt.Println("📤 Step Output Tests")

	outputYaml := filepath.Join(fixturesDir, "outputs.yaml")
	r.RunTest("step outputs captured", fmt.Sprintf("%s run %s 2>&1", r.Binary, outputYaml), "producer")
	r.RunTest("outputs passed to next step", fmt.Sprintf("%s run %s 2>&1", r.Binary, outputYaml), "consumer")

	// ========== TIMEOUT TESTS ==========
	fmt.Println("")
	fmt.Println("⏱️  Timeout Tests")

	timeoutYaml := filepath.Join(fixturesDir, "timeout.yaml")
	start := time.Now()
	r.RunTestExitCode("timeout kills long step", fmt.Sprintf("%s run %s", r.Binary, timeoutYaml), 1)
	elapsed := time.Since(start)
	if elapsed < 8*time.Second {
		fmt.Printf("  timeout respected (took %.1fs) ... ✅ PASS\n", elapsed.Seconds())
		r.Passed++
		r.Results = append(r.Results, CLITestResult{Name: "timeout respected", Passed: true})
	} else {
		fmt.Printf("  timeout respected ... ❌ FAIL (took %.1fs, expected <8s)\n", elapsed.Seconds())
		r.Failed++
		r.Results = append(r.Results, CLITestResult{Name: "timeout respected", Passed: false})
	}

	// ========== RETRY TESTS ==========
	fmt.Println("")
	fmt.Println("🔁 Retry Tests")

	// Clean up before retry test
	os.Remove("/tmp/workflow_retry_test")
	retryYaml := filepath.Join(fixturesDir, "retry.yaml")
	r.RunTestExitCode("retry succeeds on second attempt", fmt.Sprintf("%s run %s", r.Binary, retryYaml), 0)
	r.RunTestNotContains("retry shows success", fmt.Sprintf("%s run %s 2>&1", r.Binary, retryYaml), "failed")

	// ========== WORKING DIRECTORY TESTS ==========
	fmt.Println("")
	fmt.Println("📁 Working Directory Tests")

	wdYaml := filepath.Join(fixturesDir, "working_dir.yaml")
	r.RunTest("working_dir changes directory", fmt.Sprintf("%s run %s 2>&1", r.Binary, wdYaml), "check-dir")

	// ========== VALIDATION TESTS ==========
	fmt.Println("")
	fmt.Println("✅ Validation Tests")

	// Valid workflow
	r.RunTestExitCode("validate accepts valid workflow", fmt.Sprintf("%s validate %s", r.Binary, simpleYaml), 0)

	// Circular dependency
	circularYaml := filepath.Join(fixturesDir, "circular.yaml")
	r.RunTestExitCode("validate rejects circular deps", fmt.Sprintf("%s validate %s", r.Binary, circularYaml), 1)
	r.RunTest("circular error message", fmt.Sprintf("%s validate %s 2>&1", r.Binary, circularYaml), "circular")

	// Missing dependency
	missingYaml := filepath.Join(fixturesDir, "missing_dep.yaml")
	r.RunTestExitCode("validate rejects missing dep", fmt.Sprintf("%s validate %s", r.Binary, missingYaml), 1)
	r.RunTest("missing dep error message", fmt.Sprintf("%s validate %s 2>&1 | tr 'A-Z' 'a-z'", r.Binary, missingYaml), "nonexistent")

	// Invalid YAML
	invalidYaml := filepath.Join(fixturesDir, "invalid_yaml.yaml")
	r.RunTestExitCode("validate rejects invalid YAML", fmt.Sprintf("%s validate %s", r.Binary, invalidYaml), 1)

	// No steps
	noStepsYaml := filepath.Join(fixturesDir, "no_steps.yaml")
	r.RunTestExitCode("validate rejects empty workflow", fmt.Sprintf("%s validate %s", r.Binary, noStepsYaml), 1)

	// Duplicate IDs
	dupYaml := filepath.Join(fixturesDir, "duplicate_ids.yaml")
	r.RunTestExitCode("validate rejects duplicate IDs", fmt.Sprintf("%s validate %s", r.Binary, dupYaml), 1)

	// ========== ERROR HANDLING TESTS ==========
	fmt.Println("")
	fmt.Println("⚠️  Error Handling Tests")

	r.RunTestExitCode("missing file returns error", fmt.Sprintf("%s run /nonexistent/workflow.yaml", r.Binary), 1)
	r.RunTest("missing file shows error", fmt.Sprintf("%s run /nonexistent/workflow.yaml 2>&1 | tr 'A-Z' 'a-z'", r.Binary), "error")

	// ========== DRY RUN TESTS ==========
	fmt.Println("")
	fmt.Println("🏃 Dry Run Tests")

	r.RunTestExitCode("dry-run exits 0 for valid", fmt.Sprintf("%s run %s --dry-run 2>&1 || %s run --dry-run %s 2>&1", r.Binary, simpleYaml, r.Binary, simpleYaml), 0)
	// Dry run should NOT actually execute the command
	output := runCmd(projectDir, fmt.Sprintf("%s run %s --dry-run 2>&1 || %s run --dry-run %s 2>&1", r.Binary, simpleYaml, r.Binary, simpleYaml))
	if !strings.Contains(output, "Hello, World!") || strings.Contains(strings.ToLower(output), "dry") {
		fmt.Printf("  dry-run doesn't execute commands ... ✅ PASS\n")
		r.Passed++
		r.Results = append(r.Results, CLITestResult{Name: "dry-run doesn't execute commands", Passed: true})
	} else {
		fmt.Printf("  dry-run doesn't execute commands ... ❌ FAIL (command output found)\n")
		r.Failed++
		r.Results = append(r.Results, CLITestResult{Name: "dry-run doesn't execute commands", Passed: false})
	}

	// ========== LIST TESTS ==========
	fmt.Println("")
	fmt.Println("📋 List Command Tests")

	listOutput := runCmd(projectDir, fmt.Sprintf("%s list %s 2>&1", r.Binary, seqYaml))
	hasListCmd := strings.Contains(listOutput, "step1") || strings.Contains(listOutput, "Step") ||
		strings.Contains(strings.ToLower(listOutput), "unknown")
	if !strings.Contains(strings.ToLower(listOutput), "unknown") && strings.Contains(listOutput, "step") {
		fmt.Printf("  list command shows steps ... ✅ PASS\n")
		r.Passed++
		r.Results = append(r.Results, CLITestResult{Name: "list command shows steps", Passed: true})
	} else if hasListCmd {
		fmt.Printf("  list command shows steps ... ❌ FAIL (list command not implemented)\n")
		r.Failed++
		r.Results = append(r.Results, CLITestResult{Name: "list command shows steps", Passed: false})
	} else {
		fmt.Printf("  list command shows steps ... ❌ FAIL\n")
		r.Failed++
		r.Results = append(r.Results, CLITestResult{Name: "list command shows steps", Passed: false})
	}

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

// findWorkflowBuildPath finds the path to build the workflow binary
func findWorkflowBuildPath(projectDir string) string {
	// Check common locations
	locations := []string{
		filepath.Join(projectDir, "cmd", "workflow", "main.go"),
		filepath.Join(projectDir, "cmd", "workflow"),
		filepath.Join(projectDir, "main.go"),
		filepath.Join(projectDir, "."),
	}

	for _, loc := range locations {
		if fileExists(loc) {
			if strings.HasSuffix(loc, "main.go") {
				return filepath.Dir(loc)
			}
			// Check if it's a directory with main.go
			if fileExists(filepath.Join(loc, "main.go")) {
				return loc
			}
		}
	}

	// Check nested directories for cmd/workflow
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			nestedPath := filepath.Join(projectDir, entry.Name())
			if fileExists(filepath.Join(nestedPath, "cmd", "workflow", "main.go")) {
				return filepath.Join(nestedPath, "cmd", "workflow")
			}
			if fileExists(filepath.Join(nestedPath, "main.go")) {
				return nestedPath
			}
		}
	}

	return ""
}

// findWorkflowBinary finds the workflow binary after build
func findWorkflowBinary(projectDir string) string {
	locations := []string{
		filepath.Join(projectDir, "workflow"),
		filepath.Join(projectDir, "bin", "workflow"),
	}

	for _, loc := range locations {
		if fileExists(loc) {
			return loc
		}
	}

	// Check nested directories
	entries, _ := os.ReadDir(projectDir)
	for _, entry := range entries {
		if entry.IsDir() {
			nestedPath := filepath.Join(projectDir, entry.Name(), "workflow")
			if fileExists(nestedPath) {
				return nestedPath
			}
		}
	}

	// Default to ./workflow
	return "./workflow"
}

// runCmd executes a command and returns its output
func runCmd(dir, cmd string) string {
	c := exec.Command("bash", "-c", cmd)
	c.Dir = dir
	output, _ := c.CombinedOutput()
	return string(output)
}

// RunTestOutputOrder checks if strings appear in the correct order in output
func (r *CLITestRunner) RunTestOutputOrder(name, cmd string, expectedOrder []string) {
	c := exec.Command("bash", "-c", cmd)
	c.Dir = r.ProjectDir
	output, _ := c.CombinedOutput()
	outputStr := string(output)

	lastIdx := -1
	passed := true
	for _, expected := range expectedOrder {
		idx := strings.Index(outputStr, expected)
		if idx == -1 {
			passed = false
			break
		}
		if idx <= lastIdx {
			passed = false
			break
		}
		lastIdx = idx
	}

	if passed {
		fmt.Printf("  %s... ✅ PASS\n", name)
		r.Passed++
	} else {
		fmt.Printf("  %s... ❌ FAIL\n", name)
		r.Failed++
	}
	r.Results = append(r.Results, CLITestResult{Name: name, Passed: passed})
}

// RunTestNotContains checks that output does NOT contain a string
func (r *CLITestRunner) RunTestNotContains(name, cmd, notExpected string) {
	c := exec.Command("bash", "-c", cmd)
	c.Dir = r.ProjectDir
	output, _ := c.CombinedOutput()
	outputStr := string(output)

	passed := !strings.Contains(outputStr, notExpected)
	if passed {
		fmt.Printf("  %s... ✅ PASS\n", name)
		r.Passed++
	} else {
		fmt.Printf("  %s... ❌ FAIL (found: %s)\n", name, notExpected)
		r.Failed++
	}
	r.Results = append(r.Results, CLITestResult{Name: name, Passed: passed})
}
