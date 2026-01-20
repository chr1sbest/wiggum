package eval

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TestResult represents the results of running shared tests
type TestResult struct {
	Passed  int
	Failed  int
	Skipped int
	Total   int
}

// RunSharedTests executes the shared test suite for a project.
// It sets up the environment, starts the app (for web apps), runs tests, and returns the results.
func RunSharedTests(projectDir string, suite *SuiteConfig, port int) (*TestResult, error) {
	if port == 0 {
		port = 8000 // Default port
	}

	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║            Running Shared E2E Tests                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Printf("Project: %s\n", projectDir)
	fmt.Printf("Suite:   %s\n", suite.Name)
	fmt.Printf("Port:    %d\n", port)
	fmt.Println("")

	// Route based on suite type
	if suite.IsCLI() {
		return runCLITests(projectDir, suite)
	}

	// Route to Go-based API tests for specific suites
	if suite.Name == "tasktracker" {
		return runWebAPITests(projectDir, suite, port)
	}

	// Web app flow: find app, setup, start server, run pytest
	appDir, err := findAppDirectory(projectDir)
	if err != nil {
		return nil, err
	}

	// Set up Python venv if needed
	if err := setupVenv(appDir); err != nil {
		return nil, fmt.Errorf("failed to set up venv: %w", err)
	}

	// Set up .env file if needed
	if err := setupEnvFile(appDir); err != nil {
		return nil, fmt.Errorf("failed to set up .env: %w", err)
	}

	// Start Docker services if needed
	if err := startDockerServices(appDir); err != nil {
		// Don't fail if docker isn't available, just warn
		fmt.Printf("WARNING: failed to start Docker services: %v\n", err)
	}

	// Kill any existing process on the port
	if err := killProcessOnPort(port); err != nil {
		fmt.Printf("WARNING: failed to kill process on port %d: %v\n", port, err)
	}

	// Start the app in background
	appCmd, err := startApp(appDir, port)
	if err != nil {
		return nil, err
	}
	defer func() {
		if appCmd != nil && appCmd.Process != nil {
			appCmd.Process.Kill()
		}
	}()

	// Wait for app to be ready
	if err := waitForApp(port, 30*time.Second, appCmd); err != nil {
		return nil, err
	}

	// Run pytest - need absolute path since we run from project dir
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	suiteDir := filepath.Join(cwd, "evals", "suites", suite.Name)
	result, err := runPytest(appDir, suiteDir, port)
	if err != nil {
		return nil, err
	}

	fmt.Println("")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("  Results: %d passed, %d failed, %d skipped\n", result.Passed, result.Failed, result.Skipped)
	fmt.Println("═══════════════════════════════════════════════════════════════")

	return result, nil
}

// runWebAPITests runs Go-based tests for web API suites
func runWebAPITests(projectDir string, suite *SuiteConfig, port int) (*TestResult, error) {
	// Find the actual project directory (handle nested dirs)
	appDir, err := findAppDirectory(projectDir)
	if err != nil {
		return nil, err
	}

	// Set up Python venv if needed
	if err := setupVenv(appDir); err != nil {
		fmt.Printf("WARNING: failed to set up venv: %v\n", err)
	}

	// Set up .env file if needed
	if err := setupEnvFile(appDir); err != nil {
		fmt.Printf("WARNING: failed to set up .env: %v\n", err)
	}

	// Kill any existing process on the port
	if err := killProcessOnPort(port); err != nil {
		fmt.Printf("WARNING: failed to kill process on port %d: %v\n", port, err)
	}

	// Start the app in background
	appCmd, err := startApp(appDir, port)
	if err != nil {
		fmt.Printf("WARNING: failed to start app: %v\n", err)
	}
	defer func() {
		if appCmd != nil && appCmd.Process != nil {
			appCmd.Process.Kill()
		}
	}()

	// Wait for app to be ready (but continue even if it fails)
	if err := waitForApp(port, 30*time.Second, appCmd); err != nil {
		fmt.Printf("WARNING: app not ready: %v\n", err)
	}

	// Run Go-based API tests (they will fail if app isn't running)
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	switch suite.Name {
	case "tasktracker":
		return RunTasktrackerTests(baseURL)
	default:
		return nil, fmt.Errorf("no Go test runner for web API suite: %s", suite.Name)
	}
}

// runCLITests runs tests for CLI tool suites
func runCLITests(projectDir string, suite *SuiteConfig) (*TestResult, error) {
	// Find the actual project directory (handle nested dirs)
	appDir, err := findCLIAppDirectory(projectDir, suite.Language)
	if err != nil {
		return nil, err
	}

	// Get fixtures directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	fixturesDir := filepath.Join(cwd, "evals", "suites", suite.Name, "fixtures")

	// Route to appropriate test runner based on suite name
	switch suite.Name {
	case "logagg":
		return RunLogaggTests(appDir, fixturesDir)
	default:
		return nil, fmt.Errorf("no Go test runner for CLI suite: %s", suite.Name)
	}
}

// findCLIAppDirectory finds the app directory for CLI tools
func findCLIAppDirectory(projectDir string, language string) (string, error) {
	// For Go projects, look for go.mod or main.go
	if language == "go" {
		if fileExists(filepath.Join(projectDir, "go.mod")) ||
			fileExists(filepath.Join(projectDir, "main.go")) {
			return projectDir, nil
		}

		// Check nested directories
		entries, err := os.ReadDir(projectDir)
		if err != nil {
			return "", fmt.Errorf("failed to read project directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				nestedPath := filepath.Join(projectDir, entry.Name())
				if fileExists(filepath.Join(nestedPath, "go.mod")) ||
					fileExists(filepath.Join(nestedPath, "main.go")) {
					return nestedPath, nil
				}
			}
		}
	}

	// For Python CLI tools
	if language == "python" {
		if fileExists(filepath.Join(projectDir, "main.py")) ||
			fileExists(filepath.Join(projectDir, "cli.py")) {
			return projectDir, nil
		}
	}

	// Default: use project dir itself
	return projectDir, nil
}

// findAppDirectory locates the actual app directory, handling nested structures
func findAppDirectory(projectDir string) (string, error) {
	// Check if app.py, main.py, run.py exists in projectDir
	if fileExists(filepath.Join(projectDir, "app.py")) ||
		fileExists(filepath.Join(projectDir, "main.py")) ||
		fileExists(filepath.Join(projectDir, "run.py")) ||
		dirExists(filepath.Join(projectDir, "app")) {
		return projectDir, nil
	}

	// Check for nested structure (ralph approach creates nested dirs)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to read project directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			nestedPath := filepath.Join(projectDir, entry.Name())
			if fileExists(filepath.Join(nestedPath, "app.py")) ||
				fileExists(filepath.Join(nestedPath, "main.py")) ||
				fileExists(filepath.Join(nestedPath, "run.py")) ||
				dirExists(filepath.Join(nestedPath, "app")) {
				return nestedPath, nil
			}
		}
	}

	return "", fmt.Errorf("no app.py, main.py, run.py, or app/ found in %s", projectDir)
}

// setupVenv creates and sets up a Python virtual environment if requirements.txt exists
func setupVenv(appDir string) error {
	requirementsPath := filepath.Join(appDir, "requirements.txt")
	venvPath := filepath.Join(appDir, "venv")

	// Check if requirements.txt exists
	if !fileExists(requirementsPath) {
		return nil // No requirements.txt, nothing to do
	}

	// Check if venv already exists
	if dirExists(venvPath) {
		return nil // Venv already exists
	}

	fmt.Println("Setting up venv...")

	// Create venv
	cmd := exec.Command("python3", "-m", "venv", "venv")
	cmd.Dir = appDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create venv: %w", err)
	}

	// Install packages
	// We need to use the venv's pip
	pipPath := filepath.Join(venvPath, "bin", "pip")

	// Install pytest and requests
	cmd = exec.Command(pipPath, "install", "-q", "requests", "pytest")
	cmd.Dir = appDir
	if err := cmd.Run(); err != nil {
		fmt.Printf("WARNING: failed to install test dependencies: %v\n", err)
	}

	// Install requirements.txt
	cmd = exec.Command(pipPath, "install", "-q", "-r", "requirements.txt")
	cmd.Dir = appDir
	if err := cmd.Run(); err != nil {
		fmt.Printf("WARNING: failed to install requirements.txt: %v\n", err)
	}

	return nil
}

// setupEnvFile copies .env.example to .env if it exists and .env doesn't
func setupEnvFile(appDir string) error {
	envExamplePath := filepath.Join(appDir, ".env.example")
	envPath := filepath.Join(appDir, ".env")

	if fileExists(envExamplePath) && !fileExists(envPath) {
		fmt.Println("Copying .env.example to .env...")
		data, err := os.ReadFile(envExamplePath)
		if err != nil {
			return fmt.Errorf("failed to read .env.example: %w", err)
		}
		if err := os.WriteFile(envPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write .env: %w", err)
		}
	}

	return nil
}

// startDockerServices starts Docker Compose services if docker-compose.yml exists
func startDockerServices(appDir string) error {
	dockerComposePath := filepath.Join(appDir, "docker-compose.yml")
	if !fileExists(dockerComposePath) {
		return nil // No docker-compose.yml, nothing to do
	}

	fmt.Println("Starting Docker services (Redis, MinIO)...")
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = appDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start docker services: %w", err)
	}

	// Wait for services to start
	time.Sleep(3 * time.Second)

	return nil
}

// killProcessOnPort kills any process running on the specified port
func killProcessOnPort(port int) error {
	// Use lsof to find LISTENing process IDs on the port.
	// -nP avoids DNS/service-name lookups (faster and less error-prone).
	// -t prints only PIDs.
	cmd := exec.Command("lsof", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN", "-t")
	output, err := cmd.Output()
	if err != nil {
		// No process found, which is fine
		return nil
	}

	pids := strings.Fields(string(output))
	if len(pids) == 0 {
		return nil
	}

	var lastErr error
	for _, pid := range pids {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err != nil {
			// Best-effort: process may have already exited, or PID may be stale.
			lastErr = err
		}
	}

	return lastErr
}

// startApp starts the application in the background
func startApp(appDir string, port int) (*exec.Cmd, error) {
	fmt.Printf("Starting app on port %d...\n", port)

	var cmd *exec.Cmd
	venvPython := filepath.Join(appDir, "venv", "bin", "python")

	// Determine which Python to use
	pythonCmd := "python3"
	if fileExists(venvPython) {
		pythonCmd = venvPython
	}

	// Determine which file to run
	uvicornCmd := filepath.Join(appDir, "venv", "bin", "uvicorn")

	if fileExists(filepath.Join(appDir, "run.py")) {
		cmd = exec.Command(pythonCmd, "run.py")
	} else if fileExists(filepath.Join(appDir, "app.py")) {
		cmd = exec.Command(pythonCmd, "app.py")
	} else if fileExists(filepath.Join(appDir, "main.py")) {
		// FastAPI app - use uvicorn
		if fileExists(uvicornCmd) {
			cmd = exec.Command(uvicornCmd, "main:app", "--host", "0.0.0.0", "--port", strconv.Itoa(port))
		} else {
			// Try running main.py directly (may have uvicorn.run inside)
			cmd = exec.Command(pythonCmd, "main.py")
		}
	} else if fileExists(filepath.Join(appDir, "app", "main.py")) {
		// FastAPI app with app/ directory structure
		if fileExists(uvicornCmd) {
			cmd = exec.Command(uvicornCmd, "app.main:app", "--host", "0.0.0.0", "--port", strconv.Itoa(port))
		} else {
			cmd = exec.Command(pythonCmd, "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", strconv.Itoa(port))
		}
	} else {
		// Try Flask run
		flaskCmd := filepath.Join(appDir, "venv", "bin", "flask")
		if fileExists(flaskCmd) {
			cmd = exec.Command(flaskCmd, "run", "--port", strconv.Itoa(port))
			cmd.Env = append(os.Environ(), "FLASK_APP=app")
		} else {
			return nil, fmt.Errorf("no run.py, app.py, main.py, app/main.py, or flask command found")
		}
	}

	cmd.Dir = appDir

	// Capture output to a pipe so we can monitor for errors
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start app: %w", err)
	}

	// Stream output in background
	go streamOutput(stdout, "APP")
	go streamOutput(stderr, "APP-ERR")

	return cmd, nil
}

// streamOutput reads from a reader and prints lines with a prefix
func streamOutput(r io.Reader, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", prefix, scanner.Text())
	}
}

// waitForApp waits for the app to be ready by checking health endpoints
// If appCmd is provided, it also checks if the process has exited
func waitForApp(port int, timeout time.Duration, appCmd *exec.Cmd) error {
	fmt.Println("Waiting for app to start...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	urls := []string{
		fmt.Sprintf("http://localhost:%d", port),
		fmt.Sprintf("http://localhost:%d/auth/login", port),
		fmt.Sprintf("http://localhost:%d/health", port),
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Channel to detect if process exits
	processDone := make(chan error, 1)
	if appCmd != nil && appCmd.Process != nil {
		go func() {
			processDone <- appCmd.Wait()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for app to start")
		case err := <-processDone:
			return fmt.Errorf("app process exited: %v", err)
		case <-ticker.C:
			// Try each URL
			for _, url := range urls {
				resp, err := http.Get(url)
				if err == nil {
					resp.Body.Close()
					fmt.Println("App is ready!")
					return nil
				}
			}
		}
	}
}

// runPytest executes pytest and parses the results
func runPytest(appDir, suiteDir string, port int) (*TestResult, error) {
	fmt.Println("")
	fmt.Println("Running tests...")
	fmt.Println("")

	// Build the pytest command
	pytestCmd := "pytest"
	venvPytest := filepath.Join(appDir, "venv", "bin", "pytest")
	if fileExists(venvPytest) {
		pytestCmd = venvPytest
	}

	cmd := exec.Command(pytestCmd, suiteDir, "--tb=short", "-v")
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("EVAL_BASE_URL=http://localhost:%d", port))

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Print output
	fmt.Print(outputStr)

	// Parse results using regex
	result := &TestResult{}

	// Look for patterns like "5 passed", "2 failed", "1 skipped"
	passedRe := regexp.MustCompile(`(\d+)\s+passed`)
	failedRe := regexp.MustCompile(`(\d+)\s+failed`)
	skippedRe := regexp.MustCompile(`(\d+)\s+skipped`)

	if matches := passedRe.FindStringSubmatch(outputStr); len(matches) > 1 {
		result.Passed, _ = strconv.Atoi(matches[1])
	}

	if matches := failedRe.FindStringSubmatch(outputStr); len(matches) > 1 {
		result.Failed, _ = strconv.Atoi(matches[1])
	}

	if matches := skippedRe.FindStringSubmatch(outputStr); len(matches) > 1 {
		result.Skipped, _ = strconv.Atoi(matches[1])
	}

	result.Total = result.Passed + result.Failed + result.Skipped

	// Check for pytest error (command not found or execution failed)
	if err != nil && result.Total == 0 {
		return nil, fmt.Errorf("pytest execution failed: %w", err)
	}

	return result, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
