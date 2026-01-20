# Task Tracker REST API

Build a Python REST API for task management with JWT authentication.

## Models (4)

1. **User** - id, username, email, password_hash, created_at
2. **Project** - id, name, description, user_id, created_at
3. **Task** - id, title, description, status, project_id, user_id, created_at, updated_at
4. **Category** - id, name, color, user_id

## Authentication

- POST /api/auth/register - Register new user (username, email, password)
- POST /api/auth/login - Login, returns JWT token
- All other endpoints require Bearer token authentication
- Validate password strength (min 8 chars)
- Validate email format

## Endpoints (~15)

### Tasks
- GET /api/tasks - List user's tasks (supports ?status= filter)
- POST /api/tasks - Create task (title required, optional: description, status, project_id, category_id)
- GET /api/tasks/:id - Get task by ID
- PUT /api/tasks/:id - Update task
- DELETE /api/tasks/:id - Delete task

### Projects
- GET /api/projects - List user's projects
- POST /api/projects - Create project
- GET /api/projects/:id - Get project by ID
- PUT /api/projects/:id - Update project
- DELETE /api/projects/:id - Delete project

### Categories
- GET /api/categories - List user's categories
- POST /api/categories - Create category
- DELETE /api/categories/:id - Delete category

## Requirements

- Use FastAPI with uvicorn
- SQLite database (file-based, no external DB required)
- JWT tokens for authentication
- Secure password hashing with bcrypt
- Choose well-maintained libraries that work with Python 3.13
- Users can only access their own data
- Task status enum: todo, in_progress, done
- Return proper HTTP status codes (200, 201, 400, 401, 404)
- Run on port 8000
