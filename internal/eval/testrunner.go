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
// It sets up the environment, starts the app, runs tests, and returns the results.
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

	// Find the actual app directory (handle nested dirs for ralph approach)
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
	if err := waitForApp(port, 30*time.Second); err != nil {
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

// findAppDirectory locates the actual app directory, handling nested structures
func findAppDirectory(projectDir string) (string, error) {
	// Check if app.py or run.py exists in projectDir
	if fileExists(filepath.Join(projectDir, "app.py")) ||
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
				fileExists(filepath.Join(nestedPath, "run.py")) ||
				dirExists(filepath.Join(nestedPath, "app")) {
				return nestedPath, nil
			}
		}
	}

	return "", fmt.Errorf("no app.py, run.py, or app/ found in %s", projectDir)
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
	// Use lsof to find the process
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err != nil {
		// No process found, which is fine
		return nil
	}

	// Kill the process
	pid := strings.TrimSpace(string(output))
	if pid != "" {
		killCmd := exec.Command("kill", "-9", pid)
		return killCmd.Run()
	}

	return nil
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
	if fileExists(filepath.Join(appDir, "run.py")) {
		cmd = exec.Command(pythonCmd, "run.py")
	} else if fileExists(filepath.Join(appDir, "app.py")) {
		cmd = exec.Command(pythonCmd, "app.py")
	} else {
		// Try Flask run
		flaskCmd := filepath.Join(appDir, "venv", "bin", "flask")
		if fileExists(flaskCmd) {
			cmd = exec.Command(flaskCmd, "run", "--port", strconv.Itoa(port))
			cmd.Env = append(os.Environ(), "FLASK_APP=app")
		} else {
			return nil, fmt.Errorf("no run.py, app.py, or flask command found")
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
func waitForApp(port int, timeout time.Duration) error {
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

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for app to start")
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
