![drawing-board](http://counter.seku.su/cmoe?name=drawing-board&theme=rule34-big)

# Drawing Board (React + Go)

A collaborative drawing board with handwriting recognition and user accounts:
- **React (TypeScript)** canvas drawing with real-time WebSocket communication
- **Go backend** with Gorilla mux, WebSocket, SQLite persistence
- **User authentication** with session-based login/register/logout
- **Drawing tools**: Pencil and Eraser with hit-testing
- **Undo functionality**: Ctrl+Z to undo last stroke
- **Handwriting recognition**: AI-powered Japanese character recognition
- **Stroke persistence**: All drawings saved per user and restored on login

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
- **Personal Drawings**: Each user's drawings are saved separately
- **Auto-restore**: Your drawings are automatically loaded when you log in

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
1. **Register**: Create a new account with your email and password
2. **Login**: Sign in to access your personal drawing space
3. **Draw**: Use the pencil tool to draw on the canvas
4. **Save**: Your drawings are automatically saved as you draw

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
3. **View results** showing possible characters with confidence scores
4. **Try different patterns** to see how the AI recognizes various shapes

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
- `POST /api/register` - Register new user `{ email, password }`
- `POST /api/login` - Login user `{ email, password }`
- `POST /api/logout` - Logout current user
- `GET /api/me` - Get current user info

### Drawing Endpoints
- `GET /api/strokes` - Get user's saved strokes (authenticated)
- `POST /api/strokes/clear` - Clear all user's strokes (authenticated)
- `POST /api/strokes/delete?id={id}` - Delete specific stroke (authenticated)

### Recognition Endpoint
- `POST /api/recognize` - Recognize drawn characters `{ topN: 10, width: 300, height: 300 }`

### WebSocket
- `WS /ws` - Real-time drawing communication (authenticated via cookie)

**WebSocket Messages:**
```json
// Send stroke
{"type":"stroke","stroke":{"points":[{"x":10,"y":20}],"color":"#1d4ed8","width":4,"clientId":"abc","startedAtUnixMs":1690000000000}}

// Delete stroke
{"type":"delete","delete":123}
```

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
2. **Recognition not working**: Check server logs for debug output
3. **WebSocket connection failed**: Ensure backend is running on port 8080
4. **Frontend not loading**: Check if `npm run dev` is running on port 5173

### Debug Mode
The server provides detailed debug logging for recognition:
```
Recognition analysis for 2 strokes:
  Features: horizontal_lines=1.0, vertical_lines=1.0, diagonal_lines=0.0
  Patterns: has_cross=1.0, has_three_horizontal=0.0, has_two_horizontal=0.0
  Generated 2 candidates: 十(0.95), ＋(0.80)
```

## Development

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
├── web/                # React frontend
│   ├── src/           # TypeScript source
│   └── public/        # Static assets
└── models/            # ONNX model files
```

### Make Commands

#### Development Commands
```bash
make backend          # Run Go backend
make frontend         # Run React frontend  
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
