package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APITestRunner runs HTTP API tests
type APITestRunner struct {
	BaseURL string
	Token   string
	Results []CLITestResult
	Passed  int
	Failed  int
	client  *http.Client
}

// NewAPITestRunner creates a new API test runner
func NewAPITestRunner(baseURL string) *APITestRunner {
	return &APITestRunner{
		BaseURL: baseURL,
		Results: []CLITestResult{},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// doRequest makes an HTTP request and returns the response
func (r *APITestRunner) doRequest(method, path string, body interface{}) (*http.Response, map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, r.BaseURL+path, reqBody)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	var data map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	json.Unmarshal(respBody, &data)

	return resp, data, nil
}

// getStatus safely returns the status code from a response, or 0 if nil
func getStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// recordResult records a test result
func (r *APITestRunner) recordResult(name string, passed bool, msg string) {
	if passed {
		fmt.Printf("  %s... ✅ PASS\n", name)
		r.Passed++
	} else {
		fmt.Printf("  %s... ❌ FAIL\n", name)
		if msg != "" {
			fmt.Printf("    %s\n", msg)
		}
		r.Failed++
	}
	r.Results = append(r.Results, CLITestResult{Name: name, Passed: passed, Message: msg})
}

// RunTasktrackerTests runs the tasktracker API test suite
func RunTasktrackerTests(baseURL string) (*TestResult, error) {
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║            Task Tracker API Eval Suite                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Println("")

	r := NewAPITestRunner(baseURL)

	// ========== AUTH TESTS ==========
	fmt.Println("🔐 Authentication Tests")

	// Test: User registration
	testUser := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	regBody := map[string]string{
		"username": testUser,
		"email":    testUser + "@example.com",
		"password": "TestPassword123!",
	}
	resp, _, err := r.doRequest("POST", "/api/auth/register", regBody)
	if err != nil {
		r.recordResult("user registration", false, err.Error())
	} else {
		r.recordResult("user registration", getStatus(resp) == 200 || getStatus(resp) == 201,
			fmt.Sprintf("status %d", getStatus(resp)))
	}

	// Test: Login returns JWT (try both username and email formats)
	testEmail := testUser + "@example.com"
	loginBody := map[string]string{
		"username": testUser,
		"password": "TestPassword123!",
	}
	resp, data, err := r.doRequest("POST", "/api/auth/login", loginBody)
	// If username login fails with 422, try email-based login
	if resp != nil && getStatus(resp) == 422 {
		loginBody = map[string]string{
			"email":    testEmail,
			"password": "TestPassword123!",
		}
		resp, data, err = r.doRequest("POST", "/api/auth/login", loginBody)
	}
	if err != nil {
		r.recordResult("login returns JWT", false, err.Error())
	} else {
		token := ""
		if t, ok := data["token"].(string); ok {
			token = t
		} else if t, ok := data["access_token"].(string); ok {
			token = t
		} else if t, ok := data["jwt"].(string); ok {
			token = t
		}
		passed := getStatus(resp) == 200 && token != ""
		r.recordResult("login returns JWT", passed, fmt.Sprintf("status %d, token=%v", getStatus(resp), token != ""))
		r.Token = token
	}

	// Test: Invalid credentials rejected (try both formats)
	badLogin := map[string]string{
		"username": "nonexistent_user",
		"password": "WrongPassword123!",
	}
	resp, _, _ = r.doRequest("POST", "/api/auth/login", badLogin)
	if resp != nil && getStatus(resp) == 422 {
		badLogin = map[string]string{
			"email":    "nonexistent@example.com",
			"password": "WrongPassword123!",
		}
		resp, _, _ = r.doRequest("POST", "/api/auth/login", badLogin)
	}
	statusCode := 0
	if resp != nil {
		statusCode = getStatus(resp)
	}
	r.recordResult("invalid credentials rejected", resp != nil && (getStatus(resp) == 401 || getStatus(resp) == 403),
		fmt.Sprintf("status %d", statusCode))

	// Test: Protected route requires auth
	oldToken := r.Token
	r.Token = ""
	resp, _, _ = r.doRequest("GET", "/api/tasks", nil)
	r.recordResult("protected route requires auth", resp != nil && getStatus(resp) == 401,
		fmt.Sprintf("status %d", getStatus(resp)))
	r.Token = oldToken

	// Test: Invalid token rejected
	r.Token = "invalid_token_12345"
	resp, _, _ = r.doRequest("GET", "/api/tasks", nil)
	r.recordResult("invalid token rejected", resp != nil && getStatus(resp) == 401,
		fmt.Sprintf("status %d", getStatus(resp)))
	r.Token = oldToken

	// Test: Duplicate username rejected
	resp, _, _ = r.doRequest("POST", "/api/auth/register", regBody)
	r.recordResult("duplicate username rejected", resp != nil && (getStatus(resp) == 400 || getStatus(resp) == 409),
		fmt.Sprintf("status %d", getStatus(resp)))

	// ========== TASK CRUD TESTS ==========
	fmt.Println("")
	fmt.Println("📋 Task CRUD Tests")

	// Test: Create task
	taskBody := map[string]string{
		"title":       fmt.Sprintf("Test Task %d", time.Now().UnixNano()),
		"description": "This is a test task",
		"status":      "todo",
	}
	resp, data, _ = r.doRequest("POST", "/api/tasks", taskBody)
	var taskID interface{}
	if data != nil {
		taskID = data["id"]
	}
	r.recordResult("create task", resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 201) && taskID != nil,
		fmt.Sprintf("status %d, id=%v", getStatus(resp), taskID))

	// Test: List tasks
	resp, data, _ = r.doRequest("GET", "/api/tasks", nil)
	var tasks []interface{}
	if data != nil {
		if t, ok := data["tasks"].([]interface{}); ok {
			tasks = t
		} else if t, ok := data["data"].([]interface{}); ok {
			tasks = t
		} else if t, ok := data["items"].([]interface{}); ok {
			tasks = t
		}
	}
	r.recordResult("list tasks", resp != nil && getStatus(resp) == 200,
		fmt.Sprintf("status %d, count=%d", getStatus(resp), len(tasks)))

	// Test: Get task by ID
	if taskID != nil {
		resp, data, _ = r.doRequest("GET", fmt.Sprintf("/api/tasks/%v", taskID), nil)
		r.recordResult("get task by ID", resp != nil && getStatus(resp) == 200 && data["id"] == taskID,
			fmt.Sprintf("status %d", getStatus(resp)))
	} else {
		r.recordResult("get task by ID", false, "no task ID from create")
	}

	// Test: Update task
	if taskID != nil {
		updateBody := map[string]string{
			"title":       "Updated Title",
			"description": "Updated description",
			"status":      "done",
		}
		resp, data, _ = r.doRequest("PUT", fmt.Sprintf("/api/tasks/%v", taskID), updateBody)
		if resp != nil && getStatus(resp) == 405 {
			resp, data, _ = r.doRequest("PATCH", fmt.Sprintf("/api/tasks/%v", taskID), updateBody)
		}
		passed := resp != nil && getStatus(resp) == 200 && data["title"] == "Updated Title"
		r.recordResult("update task", passed, fmt.Sprintf("status %d", getStatus(resp)))
	} else {
		r.recordResult("update task", false, "no task ID")
	}

	// Test: Create task with minimal data
	minimalTask := map[string]string{"title": fmt.Sprintf("Minimal Task %d", time.Now().UnixNano())}
	resp, data, _ = r.doRequest("POST", "/api/tasks", minimalTask)
	r.recordResult("create task with minimal data", resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 201),
		fmt.Sprintf("status %d", getStatus(resp)))

	// Test: Create task without title fails
	noTitle := map[string]string{"description": "No title task"}
	resp, _, _ = r.doRequest("POST", "/api/tasks", noTitle)
	r.recordResult("create task without title fails", resp != nil && getStatus(resp) == 400,
		fmt.Sprintf("status %d", getStatus(resp)))

	// Test: Delete task
	if taskID != nil {
		resp, _, _ = r.doRequest("DELETE", fmt.Sprintf("/api/tasks/%v", taskID), nil)
		r.recordResult("delete task", resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 204),
			fmt.Sprintf("status %d", getStatus(resp)))

		// Verify deleted
		resp, _, _ = r.doRequest("GET", fmt.Sprintf("/api/tasks/%v", taskID), nil)
		r.recordResult("deleted task returns 404", resp != nil && getStatus(resp) == 404,
			fmt.Sprintf("status %d", getStatus(resp)))
	} else {
		r.recordResult("delete task", false, "no task ID")
		r.recordResult("deleted task returns 404", false, "no task ID")
	}

	// Test: Invalid task ID
	resp, _, _ = r.doRequest("GET", "/api/tasks/999999", nil)
	r.recordResult("invalid task ID returns 404", resp != nil && getStatus(resp) == 404,
		fmt.Sprintf("status %d", getStatus(resp)))

	// ========== FILTER TESTS ==========
	fmt.Println("")
	fmt.Println("🔍 Filter Tests")

	// Create tasks with different statuses for filter tests
	for _, status := range []string{"todo", "in_progress", "done"} {
		body := map[string]string{
			"title":  fmt.Sprintf("Filter Test %s %d", status, time.Now().UnixNano()),
			"status": status,
		}
		r.doRequest("POST", "/api/tasks", body)
	}

	// Test: Filter by status
	resp, data, _ = r.doRequest("GET", "/api/tasks?status=todo", nil)
	if resp != nil && getStatus(resp) == 200 {
		r.recordResult("filter tasks by status", true, "")
	} else if resp != nil && getStatus(resp) == 400 {
		r.recordResult("filter tasks by status", false, "filtering not supported")
	} else {
		r.recordResult("filter tasks by status", false, fmt.Sprintf("status %d", getStatus(resp)))
	}

	// ========== PROJECT TESTS ==========
	fmt.Println("")
	fmt.Println("📁 Project Tests")

	// Test: Create project
	projectBody := map[string]string{
		"name":        fmt.Sprintf("Test Project %d", time.Now().UnixNano()),
		"description": "A test project",
	}
	resp, data, _ = r.doRequest("POST", "/api/projects", projectBody)
	var projectID interface{}
	if data != nil {
		projectID = data["id"]
	}
	if resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 201) {
		r.recordResult("create project", true, "")
	} else if resp != nil && getStatus(resp) == 404 {
		r.recordResult("create project", false, "projects endpoint not found")
	} else {
		r.recordResult("create project", false, fmt.Sprintf("status %d", getStatus(resp)))
	}

	// Test: List projects
	resp, _, _ = r.doRequest("GET", "/api/projects", nil)
	if resp != nil && getStatus(resp) == 200 {
		r.recordResult("list projects", true, "")
	} else if resp != nil && getStatus(resp) == 404 {
		r.recordResult("list projects", false, "projects endpoint not found")
	} else {
		r.recordResult("list projects", false, fmt.Sprintf("status %d", getStatus(resp)))
	}

	// Test: Get project by ID
	if projectID != nil {
		resp, _, _ = r.doRequest("GET", fmt.Sprintf("/api/projects/%v", projectID), nil)
		r.recordResult("get project by ID", resp != nil && getStatus(resp) == 200, fmt.Sprintf("status %d", getStatus(resp)))
	} else {
		r.recordResult("get project by ID", false, "no project ID")
	}

	// Test: Create task in project
	if projectID != nil {
		taskInProject := map[string]interface{}{
			"title":      fmt.Sprintf("Project Task %d", time.Now().UnixNano()),
			"project_id": projectID,
		}
		resp, data, _ = r.doRequest("POST", "/api/tasks", taskInProject)
		hasProject := false
		if data != nil {
			if pid, ok := data["project_id"]; ok && pid == projectID {
				hasProject = true
			}
			if proj, ok := data["project"].(map[string]interface{}); ok && proj["id"] == projectID {
				hasProject = true
			}
		}
		r.recordResult("create task in project", resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 201),
			fmt.Sprintf("status %d, has_project=%v", getStatus(resp), hasProject))
	} else {
		r.recordResult("create task in project", false, "no project ID")
	}

	// Test: Delete project
	if projectID != nil {
		resp, _, _ = r.doRequest("DELETE", fmt.Sprintf("/api/projects/%v", projectID), nil)
		r.recordResult("delete project", resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 204),
			fmt.Sprintf("status %d", getStatus(resp)))
	} else {
		r.recordResult("delete project", false, "no project ID")
	}

	// ========== CATEGORY/TAG TESTS ==========
	fmt.Println("")
	fmt.Println("🏷️  Category/Tag Tests")

	// Test: Create category
	catBody := map[string]string{"name": fmt.Sprintf("Test Category %d", time.Now().UnixNano())}
	resp, data, _ = r.doRequest("POST", "/api/categories", catBody)
	var categoryID interface{}
	endpoint := "/api/categories"
	if resp != nil && getStatus(resp) == 404 {
		resp, data, _ = r.doRequest("POST", "/api/tags", catBody)
		endpoint = "/api/tags"
	}
	if data != nil {
		categoryID = data["id"]
	}
	if resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 201) {
		r.recordResult("create category/tag", true, "")
	} else {
		r.recordResult("create category/tag", false, fmt.Sprintf("status %d", getStatus(resp)))
	}

	// Test: List categories
	resp, _, _ = r.doRequest("GET", endpoint, nil)
	if resp != nil && getStatus(resp) == 200 {
		r.recordResult("list categories/tags", true, "")
	} else {
		r.recordResult("list categories/tags", false, fmt.Sprintf("status %d", getStatus(resp)))
	}

	// Test: Delete category
	if categoryID != nil {
		resp, _, _ = r.doRequest("DELETE", fmt.Sprintf("%s/%v", endpoint, categoryID), nil)
		r.recordResult("delete category/tag", resp != nil && (getStatus(resp) == 200 || getStatus(resp) == 204),
			fmt.Sprintf("status %d", getStatus(resp)))
	} else {
		r.recordResult("delete category/tag", false, "no category ID")
	}

	// ========== VALIDATION TESTS ==========
	fmt.Println("")
	fmt.Println("⚠️  Validation Tests")

	// Test: Weak password rejected
	weakPwdBody := map[string]string{
		"username": fmt.Sprintf("weakpwd_%d", time.Now().UnixNano()),
		"email":    fmt.Sprintf("weakpwd_%d@example.com", time.Now().UnixNano()),
		"password": "123",
	}
	resp, _, _ = r.doRequest("POST", "/api/auth/register", weakPwdBody)
	r.recordResult("weak password rejected", resp != nil && getStatus(resp) == 400,
		fmt.Sprintf("status %d", getStatus(resp)))

	// Test: Invalid email rejected
	badEmailBody := map[string]string{
		"username": fmt.Sprintf("bademail_%d", time.Now().UnixNano()),
		"email":    "not-an-email",
		"password": "ValidPassword123!",
	}
	resp, _, _ = r.doRequest("POST", "/api/auth/register", badEmailBody)
	r.recordResult("invalid email rejected", resp != nil && getStatus(resp) == 400,
		fmt.Sprintf("status %d", getStatus(resp)))

	// Test: Empty title rejected
	emptyTitle := map[string]string{"title": "", "description": "Empty title"}
	resp, _, _ = r.doRequest("POST", "/api/tasks", emptyTitle)
	r.recordResult("empty title rejected", resp != nil && getStatus(resp) == 400,
		fmt.Sprintf("status %d", getStatus(resp)))

	// ========== USER ISOLATION TESTS ==========
	fmt.Println("")
	fmt.Println("🔒 User Isolation Tests")

	// Create a task as user 1
	isolationTask := map[string]string{"title": fmt.Sprintf("Private Task %d", time.Now().UnixNano())}
	resp, data, _ = r.doRequest("POST", "/api/tasks", isolationTask)
	var privateTaskID interface{}
	if data != nil {
		privateTaskID = data["id"]
	}

	// Create user 2 and try to access user 1's task
	user2 := fmt.Sprintf("otheruser_%d", time.Now().UnixNano())
	user2Body := map[string]string{
		"username": user2,
		"email":    user2 + "@example.com",
		"password": "OtherPassword123!",
	}
	r.doRequest("POST", "/api/auth/register", user2Body)

	// Login as user 2 (try both username and email formats)
	user2Email := user2 + "@example.com"
	user2Login := map[string]string{"username": user2, "password": "OtherPassword123!"}
	loginResp, loginData, _ := r.doRequest("POST", "/api/auth/login", user2Login)
	if loginResp != nil && loginResp.StatusCode == 422 {
		user2Login = map[string]string{"email": user2Email, "password": "OtherPassword123!"}
		_, loginData, _ = r.doRequest("POST", "/api/auth/login", user2Login)
	}
	user2Token := ""
	if loginData != nil {
		if t, ok := loginData["token"].(string); ok {
			user2Token = t
		} else if t, ok := loginData["access_token"].(string); ok {
			user2Token = t
		}
	}

	// Try to access user 1's task as user 2
	if privateTaskID != nil && user2Token != "" {
		oldToken := r.Token
		r.Token = user2Token
		resp, _, _ = r.doRequest("GET", fmt.Sprintf("/api/tasks/%v", privateTaskID), nil)
		r.recordResult("user cannot access other user's tasks",
			resp != nil && (getStatus(resp) == 403 || getStatus(resp) == 404),
			fmt.Sprintf("status %d", getStatus(resp)))
		r.Token = oldToken
	} else {
		r.recordResult("user cannot access other user's tasks", false, "setup failed")
	}

	// ========== SUMMARY ==========
	fmt.Println("")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("  Results: %d passed, %d failed out of %d\n", r.Passed, r.Failed, r.Passed+r.Failed)
	fmt.Println("═══════════════════════════════════════════════════════════════")

	return &TestResult{
		Passed: r.Passed,
		Failed: r.Failed,
		Total:  r.Passed + r.Failed,
	}, nil
}
