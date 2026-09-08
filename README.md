![drawing-board](http://counter.seku.su/cmoe?name=drawing-board&theme=rule34-big)

# Japanese Handwriting Practice (Vue + Go)

A **personal** Japanese handwriting training app: practice on a private canvas, persist your strokes, and get recognition feedback. Strokes are never shared with other users.

- **Vue 3 (TypeScript, Vite)** practice canvas with authenticated WebSocket persist/echo
- **Go backend** with Gorilla mux, WebSocket, SQLite persistence
- **User authentication** with session-based login/register/logout
- **Drawing tools**: Pencil and Eraser with hit-testing
- **Undo functionality**: Ctrl+Z to undo last stroke
- **Handwriting recognition**: AI-powered Japanese character recognition (training feedback loop)
- **Private stroke persistence**: drawings are scoped per user and restored only for that account

## Features

### Drawing Tools
- **Pencil**: Draw with customizable color and width
- **Eraser**: Remove individual strokes by clicking on them
- **Undo**: Press `Ctrl+Z` (or `Cmd+Z` on Mac) to undo the last stroke
- **Clear**: Remove all your drawings from the canvas and database

### Handwriting Recognition
- **AI Recognition**: Advanced pattern-based recognition for Japanese characters
- **Supported Characters**: 一, 二, 三, 十, 丨, 丶, 人, 大, 小, 中, 国, 学, 生, and more
- **Real-time Analysis**: Click "Recognize" to get character suggestions with confidence scores
- **Pattern Detection**: Automatically detects crosses (十), horizontal lines (三), and other patterns

### User Management
- **Registration**: Create new accounts with email and password
- **Login/Logout**: Secure session-based authentication
- **Private practice**: Each user's strokes stay private (REST + WebSocket are per-user)
- **Auto-restore**: Your practice strokes load automatically when you log in

## Requirements

### For Local Development
- **Go 1.22+**
- **Node 18+**
- **Modern web browser** with Canvas and WebSocket support

### For Docker Deployment
- **Docker 20.10+**
- **Docker Compose 2.0+**

## Quick Start

### Option 1: Docker (Recommended)

#### Production Deployment
```bash
# Clone the repository
git clone <repository-url>
cd drawing-board

# Build and run with Docker Compose
make docker-build
make docker-run

# Or use docker-compose directly
docker-compose up -d
```

The application will be available at:
- **Frontend**: http://localhost
- **Backend API**: http://localhost:8080

#### Development with Docker
```bash
# Run in development mode with hot reload
make docker-run-dev

# Or use docker-compose directly
docker-compose -f docker-compose.dev.yml up -d
```

#### Docker Management Commands
```bash
# View logs
make docker-logs
make docker-logs-backend
make docker-logs-frontend

# Stop containers
make docker-stop

# Clean up (removes containers, volumes, and images)
make docker-clean

# Access container shells
make docker-shell-backend
make docker-shell-frontend
```

### Option 2: Local Development

#### 1. Clone and Setup
```bash
git clone <repository-url>
cd drawing-board
go mod tidy
```

#### 2. Install Frontend Dependencies
```bash
cd web
npm install
cd ..
```

#### 3. Run Development Servers

**Terminal 1 - Backend:**
```bash
# Basic setup (uses Simple Recognizer)
go run ./cmd/server

# With ONNX model (advanced recognition)
ONNX_MODEL=./models/handwriting.onnx go run ./cmd/server
```

**Terminal 2 - Frontend:**
```bash
cd web
npm run dev
```

### 4. Open the Application
- **Frontend**: http://localhost:5173
- **Backend API**: http://localhost:8080

## Usage Guide

### Getting Started
1. **Open the app** — unauthenticated visits land on the public auth page (`/#/login`, or `/#/register` to create an account).
2. **Create account or sign in** — email and password (minimum 8 characters). A session cookie (`sid`) is set; the client sends `credentials: 'include'`.
3. **Practice** — after auth you are redirected to the private board. Use the pencil tool to write characters on the canvas.
4. **Save** — strokes are persisted for your account as you draw (WebSocket echo assigns server ids).
5. **Logout** — use Logout on the board to clear the session and return to the auth page.

Registration and login are **not** on the board header; they live only on the public auth routes.

### Drawing Tools
- **Color Picker**: Choose any color for your pencil
- **Width Slider**: Adjust line thickness from 1-20 pixels
- **Pencil Tool**: Default drawing tool
- **Eraser Tool**: Click on any stroke to remove it
- **Undo Button**: Click to undo the last stroke (or use Ctrl+Z)
- **Clear Button**: Remove all your drawings

### Handwriting Recognition
1. **Draw a character** on the canvas (try 一, 二, 三, 十)
2. **Click "Recognize"** button
3. **View results** showing possible characters with heuristic match scores (not calibrated confidence)
4. **Try different patterns** to see how the recognizer ranks candidates

### Keyboard Shortcuts
- **Ctrl+Z** (Windows/Linux) or **Cmd+Z** (Mac): Undo last stroke
- **Escape**: Cancel current drawing operation

## Advanced Setup

### Environment Variables
```bash
# Database configuration
DB_PATH=file:data.db?_fk=1

# Server configuration  
ADDR=:8080

# Security (change this in production!)
COOKIE_KEY=please-change-this-32-bytes-min

# ONNX model for advanced recognition
ONNX_MODEL=./models/handwriting.onnx
```

### Production Build
```bash
# Build frontend
make build-web

# Run production server
ADDR=:8080 STATIC_DIR=web/dist DB_PATH=file:data.db?_fk=1 COOKIE_KEY=your-secure-key make run
```

### ONNX Model Setup (Optional)
For advanced handwriting recognition:
```bash
# Download ONNX model (optional - uses Simple Recognizer by default)
make onnx-model

# Run with ONNX model
ONNX_MODEL=./models/handwriting.onnx go run ./cmd/server
```

## API Reference

### Authentication Endpoints
Cookie session name: `sid` (`HttpOnly`, `SameSite=Lax`, `Path=/`). Auth handlers log with prefixes `[auth.Register]`, `[auth.Login]`, `[auth.Logout]`, `[auth.Me]` (level filtered via `LOG_LEVEL`).

- `POST /api/register` — Create account `{ email, password }` (password min 8). Success `200` `{ id, email }` + session cookie.
- `POST /api/login` — Sign in `{ email, password }`. Success `200` `{ id, email }` + session cookie.
- `POST /api/logout` — Clear session cookie. Success `200` `{ "ok": "true" }`.
- `GET /api/me` — Current user or `401` `{ "error": "unauthorized" }`.

Auth error JSON shape: `{ "error": "<code>", "message": "<optional>" }`.

| HTTP | Code | Meaning |
|------|------|---------|
| 400 | `bad_json` | Body not JSON |
| 400 | `missing_fields` | Empty email or password |
| 400 | `invalid_email` | Email fails basic format check |
| 400 | `password_too_short` | Password shorter than 8 characters |
| 400 | `registration_failed` | Unable to create account (includes duplicate email; does **not** return `email exists`) |
| 401 | `invalid_credentials` | Login failed (unknown email or wrong password — same response) |
| 401 | `unauthorized` | `/api/me` without a valid session |

Password hashing is currently unsalted SHA-256; stronger KDF migration (Argon2id/bcrypt) is a follow-up hardening item. Do not treat this as production-grade password storage yet.

### Drawing Endpoints
- `GET /api/strokes` - Get user's saved strokes (authenticated)
- `POST /api/strokes/clear` - Clear all user's strokes (authenticated)
- `POST /api/strokes/delete?id={id}` - Delete specific stroke (authenticated)

### Recognition Endpoint
- `POST /api/recognize` - Recognize drawn characters `{ topN: 10, width: 300, height: 300 }`

### WebSocket
- `WS /ws` - Authenticated **private persist + echo** channel (cookie session required)

The hub delivers messages only to connections belonging to the same `user_id` (multi-tab same account receives echoes; other users never see your strokes). This is **not** a collaborative/shared board.

**WebSocket Messages:**
```json
// Send stroke (saved for the authenticated user, then echoed to that user's connections)
{"type":"stroke","stroke":{"points":[{"x":10,"y":20}],"color":"#1d4ed8","width":4,"clientId":"abc","startedAtUnixMs":1690000000000}}

// Delete stroke (scoped to the authenticated user, echoed to that user's connections)
{"type":"delete","delete":123}
```

The frontend treats unmatched inbound strokes as non-authoritative (ignores foreign live strokes) and only merges echoes that match a pending local stroke.
## Recognition System

The application includes two recognition systems:

### 1. Simple Recognizer (Default)
- **Pattern-based analysis** of stroke shapes and directions
- **No external dependencies** - works out of the box
- **Supports basic characters**: 一, 二, 三, 十, 丨, 丶, 人, 大, 小, 中, 国, 学, 生
- **Real-time analysis** with confidence scores

### 2. ONNX Recognizer (Advanced)
- **Machine learning-based** recognition using ONNX models
- **Higher accuracy** for complex characters
- **Requires ONNX model file** (see setup instructions)
- **Fallback to Simple Recognizer** if model not available

## Troubleshooting

### Common Issues
1. **"Address already in use"**: Stop existing server processes with `pkill -f "go run"`
2. **Recognition not working**: Check server logs for recognition debug output
3. **WebSocket connection failed**: Ensure backend is running on port 8080 and you are signed in
4. **Frontend not loading**: Check if `npm run dev` is running on port 5173
5. **Seeing another user's strokes**: Should not happen; verify you are on a build with per-user `sendToUser` (not global broadcast) and check logs below

### Debug Mode

Verbose server log prefixes for privacy and persistence:

| Prefix | Meaning |
|--------|---------|
| `[ws.Handle]` | Connect/disconnect (`userID`, remote), inbound stroke/delete, save/delete INFO, upgrade/read errors |
| `[ws.sendToUser]` | Delivery to one user's connections; `recipients=` should stay within that account (e.g. 1–N tabs) |
| `[httpapi.ListStrokes]` / `[httpapi.ClearStrokes]` / `[httpapi.DeleteStroke]` | Authenticated REST entry (`userID`), clear count, delete success, store errors |

Example privacy check while two users practice: user A's stroke logs should show `sendToUser` recipient counts only for A's open tabs, never B's.

Frontend (Vite dev): browser console uses `[wsClient]` and `[BoardPage.ws]` for connect/send/ignore reasons.

Recognition still logs feature/candidate analysis:
```
Recognition analysis for 2 strokes:
  Features: horizontal_lines=1.0, vertical_lines=1.0, diagonal_lines=0.0
  Patterns: has_cross=1.0, has_three_horizontal=0.0, has_two_horizontal=0.0
  Generated 2 candidates: 十(0.95), ＋(0.80)
```

## Development

### Speckit Workflow (Cursor)

The repository is initialized for **Speckit** in Cursor. Use slash commands in
Cursor chat:

```text
/speckit.constitution
/speckit.specify <feature description>
/speckit.clarify
/speckit.plan
/speckit.tasks
/speckit.implement
```

Generated project artifacts live in `.specify/`, and Cursor commands are in
`.cursor/commands/`.

### Project Structure
```
drawing-board/
├── cmd/server/          # Go backend server
├── internal/            # Go internal packages
│   ├── auth/           # Authentication logic
│   ├── db/             # Database layer
│   ├── httpapi/        # HTTP API handlers
│   ├── recognize/      # Recognition algorithms
│   └── ws/             # WebSocket handling
├── web/                # Vue 3 frontend
│   ├── src/           # TypeScript / Vue source
│   └── public/        # Static assets
└── models/            # ONNX model files
```

### Make Commands

#### Development Commands
```bash
make backend          # Run Go backend
make frontend         # Run Vue frontend
make build-web        # Build frontend for production
make run              # Run production server
make onnx-model       # Download ONNX model
make test             # Run all unit tests
make test-verbose     # Run tests with verbose output
```

#### Docker Commands
```bash
# Build containers
make docker-build           # Build all containers
make docker-build-backend   # Build backend container only
make docker-build-frontend  # Build frontend container only

# Run containers
make docker-run            # Run production containers
make docker-run-dev        # Run development containers

# Manage containers
make docker-stop           # Stop production containers
make docker-stop-dev       # Stop development containers
make docker-logs           # View all container logs
make docker-logs-backend   # View backend logs only
make docker-logs-frontend  # View frontend logs only
make docker-clean          # Remove containers, volumes, and images

# Debug containers
make docker-shell-backend  # Access backend container shell
make docker-shell-frontend # Access frontend container shell
```

## Docker Configuration

### Production Setup
The production Docker setup includes:
- **Multi-stage builds** for optimized image sizes
- **Nginx reverse proxy** for the frontend
- **Health checks** for both services
- **Persistent volumes** for database storage
- **Security headers** and optimizations

### Development Setup
The development setup provides:
- **Hot reload** capabilities
- **Volume mounting** for live code changes
- **Separate networks** for isolation
- **Debug-friendly** configuration

### Environment Variables
```bash
# Backend environment variables
PORT=8080                    # Server port
DB_PATH=/data/drawing-board.db  # Database file path
SESSION_SECRET=your-secret-key  # Session encryption key
ONNX_MODEL=./models/handwriting.onnx  # ONNX model path (optional)
```

## License
MIT License - see LICENSE file for details.
