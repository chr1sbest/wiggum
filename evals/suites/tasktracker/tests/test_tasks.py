"""
CRUD tests for Task Tracker API tasks endpoint.

These tests validate task creation, retrieval, updating, and deletion operations.
Tests are implementation-agnostic and focus on the API contract.
"""

import pytest
import uuid


def test_create_task(api_client):
    """Test that authenticated users can create tasks."""
    task_data = {
        "title": f"Test Task {uuid.uuid4().hex[:8]}",
        "description": "This is a test task",
        "status": "todo"
    }

    response = api_client.post("/api/tasks", json=task_data)

    assert response.status_code in [200, 201], \
        f"Task creation failed: {response.status_code} {response.text}"

    data = response.json()
    assert "id" in data, "Response should include task ID"
    assert data["title"] == task_data["title"]

    # Store task_id for potential cleanup
    return data.get("id")


def test_list_tasks(api_client):
    """Test that authenticated users can list their tasks."""
    # Create a task first to ensure there's at least one
    task_data = {
        "title": f"List Test Task {uuid.uuid4().hex[:8]}",
        "description": "Task for list test"
    }
    api_client.post("/api/tasks", json=task_data)

    # Now list tasks
    response = api_client.get("/api/tasks")

    assert response.status_code == 200, \
        f"List tasks failed: {response.status_code} {response.text}"

    data = response.json()

    # Response should be a list or contain a list
    if isinstance(data, dict):
        tasks = data.get("tasks") or data.get("data") or data.get("items")
        assert tasks is not None, f"Could not find tasks list in response: {data.keys()}"
    else:
        tasks = data

    assert isinstance(tasks, list), "Tasks should be a list"
    assert len(tasks) > 0, "Should have at least one task"


def test_get_task_by_id(api_client):
    """Test that authenticated users can retrieve a specific task by ID."""
    # Create a task first
    task_data = {
        "title": f"Get Test Task {uuid.uuid4().hex[:8]}",
        "description": "Task for get test",
        "status": "in_progress"
    }
    create_response = api_client.post("/api/tasks", json=task_data)
    assert create_response.status_code in [200, 201]

    task_id = create_response.json()["id"]

    # Now get the specific task
    response = api_client.get(f"/api/tasks/{task_id}")

    assert response.status_code == 200, \
        f"Get task by ID failed: {response.status_code} {response.text}"

    data = response.json()
    assert data["id"] == task_id
    assert data["title"] == task_data["title"]
    assert data["description"] == task_data["description"]


def test_update_task(api_client):
    """Test that authenticated users can update their tasks."""
    # Create a task first
    task_data = {
        "title": f"Update Test Task {uuid.uuid4().hex[:8]}",
        "description": "Original description",
        "status": "todo"
    }
    create_response = api_client.post("/api/tasks", json=task_data)
    assert create_response.status_code in [200, 201]

    task_id = create_response.json()["id"]

    # Update the task
    updated_data = {
        "title": "Updated Title",
        "description": "Updated description",
        "status": "done"
    }
    response = api_client.put(f"/api/tasks/{task_id}", json=updated_data)

    # Accept both PUT and PATCH
    if response.status_code == 405:
        response = api_client.patch(f"/api/tasks/{task_id}", json=updated_data)

    assert response.status_code == 200, \
        f"Update task failed: {response.status_code} {response.text}"

    data = response.json()
    assert data["title"] == updated_data["title"]
    assert data["description"] == updated_data["description"]
    assert data["status"] == updated_data["status"]


def test_delete_task(api_client):
    """Test that authenticated users can delete their tasks."""
    # Create a task first
    task_data = {
        "title": f"Delete Test Task {uuid.uuid4().hex[:8]}",
        "description": "Task to be deleted"
    }
    create_response = api_client.post("/api/tasks", json=task_data)
    assert create_response.status_code in [200, 201]

    task_id = create_response.json()["id"]

    # Delete the task
    response = api_client.delete(f"/api/tasks/{task_id}")

    assert response.status_code in [200, 204], \
        f"Delete task failed: {response.status_code} {response.text}"

    # Verify the task is deleted by trying to get it
    get_response = api_client.get(f"/api/tasks/{task_id}")
    assert get_response.status_code == 404, \
        "Deleted task should return 404"


def test_filter_tasks_by_status(api_client):
    """Test that tasks can be filtered by status."""
    # Create tasks with different statuses
    unique_prefix = uuid.uuid4().hex[:8]

    statuses = ["todo", "in_progress", "done"]
    task_ids = []

    for status in statuses:
        task_data = {
            "title": f"Filter Test {unique_prefix} {status}",
            "description": f"Task with status {status}",
            "status": status
        }
        response = api_client.post("/api/tasks", json=task_data)
        assert response.status_code in [200, 201]
        task_ids.append(response.json()["id"])

    # Filter by status - try common query parameter formats
    for status in statuses:
        response = api_client.get(f"/api/tasks?status={status}")

        # If query param not supported, skip this test
        if response.status_code == 400:
            pytest.skip("Status filtering not supported by this implementation")

        assert response.status_code == 200, \
            f"Filter by status failed: {response.status_code} {response.text}"

        data = response.json()

        # Extract tasks list from response
        if isinstance(data, dict):
            tasks = data.get("tasks") or data.get("data") or data.get("items")
        else:
            tasks = data

        # Verify all returned tasks have the requested status
        if tasks:  # Only check if tasks are returned
            for task in tasks:
                if task["title"].startswith(f"Filter Test {unique_prefix}"):
                    assert task["status"] == status, \
                        f"Task has wrong status: {task['status']} != {status}"


def test_task_belongs_to_user(api_client, base_url):
    """Test that users can only access their own tasks."""
    # Create a task with the first user
    task_data = {
        "title": f"Private Task {uuid.uuid4().hex[:8]}",
        "description": "This task should be private"
    }
    create_response = api_client.post("/api/tasks", json=task_data)
    assert create_response.status_code in [200, 201]

    task_id = create_response.json()["id"]

    # Create a second user and get their token
    import requests
    test_id = str(uuid.uuid4())[:8]
    username = f"otheruser_{test_id}"
    password = "OtherPassword123!"

    # Register second user
    reg_response = requests.post(
        f"{base_url}/api/auth/register",
        json={
            "username": username,
            "email": f"{username}@example.com",
            "password": password
        }
    )

    # Login as second user
    login_response = requests.post(
        f"{base_url}/api/auth/login",
        json={
            "username": username,
            "password": password
        }
    )
    assert login_response.status_code == 200

    other_token = login_response.json().get("token") or \
                  login_response.json().get("access_token") or \
                  login_response.json().get("jwt")

    # Try to access the first user's task as the second user
    response = requests.get(
        f"{base_url}/api/tasks/{task_id}",
        headers={"Authorization": f"Bearer {other_token}"}
    )

    # Should return 404 (not found) or 403 (forbidden)
    assert response.status_code in [403, 404], \
        f"User should not access other user's tasks, got {response.status_code}"


def test_invalid_task_id_handling(api_client):
    """Test that requests with invalid task IDs are handled properly."""
    # Test with non-existent numeric ID
    response = api_client.get("/api/tasks/999999")
    assert response.status_code == 404, \
        f"Expected 404 for non-existent task, got {response.status_code}"

    # Test with invalid ID format (if using UUIDs)
    response = api_client.get("/api/tasks/invalid-id-format")
    assert response.status_code in [400, 404], \
        f"Expected 400/404 for invalid task ID format, got {response.status_code}"


def test_create_task_with_minimal_data(api_client):
    """Test that tasks can be created with only required fields."""
    # Create task with only title (minimum required field)
    task_data = {
        "title": f"Minimal Task {uuid.uuid4().hex[:8]}"
    }

    response = api_client.post("/api/tasks", json=task_data)

    assert response.status_code in [200, 201], \
        f"Task creation with minimal data failed: {response.status_code} {response.text}"

    data = response.json()
    assert "id" in data
    assert data["title"] == task_data["title"]

    # Default status should be set (typically "todo" or similar)
    assert "status" in data, "Task should have a default status"


def test_create_task_without_title_fails(api_client):
    """Test that task creation without required title field fails."""
    task_data = {
        "description": "Task without title",
        "status": "todo"
    }

    response = api_client.post("/api/tasks", json=task_data)

    # Should return 400 Bad Request for missing required field
    assert response.status_code == 400, \
        f"Expected 400 for missing title, got {response.status_code}"
