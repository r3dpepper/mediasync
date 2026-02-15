# Private Media Ecosystem - Server Build Guide

## Prerequisites

- Go 1.21 or later
- SQLite 3.43 or later
- GCC (for CGO dependencies)

## Quick Build

### macOS/Linux

```bash
# Navigate to server directory
cd server

# Download dependencies
go mod download

# Build binary
go build -o media-server cmd/server/main.go

# Run
./media-server --help
```

### Windows

```powershell
cd server
go mod download
go build -o media-server.exe cmd\server\main.go
.\media-server.exe --help
```

## Using Makefile

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Clean build artifacts
make clean

# Install to system
make install
```

## Cross-Compilation

### Build for Linux from macOS

```bash
GOOS=linux GOARCH=amd64 go build -o media-server-linux cmd/server/main.go
```

### Build for macOS from Linux

```bash
GOOS=darwin GOARCH=amd64 go build -o media-server-macos cmd/server/main.go
```

### Build for Windows from Linux/macOS

```bash
GOOS=windows GOARCH=amd64 go build -o media-server.exe cmd/server/main.go
```

## Build Flags

### Release Build (Optimized)

```bash
go build -ldflags="-s -w" -o media-server cmd/server/main.go
```

Flags:
- `-s`: Strip symbol table
- `-w`: Strip DWARF debugging info

### Debug Build

```bash
go build -gcflags="all=-N -l" -o media-server cmd/server/main.go
```

## Docker Build (Optional)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o media-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /root/
COPY --from=builder /build/media-server .
EXPOSE 8080
CMD ["./media-server", "start"]
```

## Troubleshooting

### CGO Errors

If you get CGO errors during build:

**macOS:**
```bash
xcode-select --install
```

**Linux (Debian/Ubuntu):**
```bash
sudo apt-get install build-essential
```

**Windows:**
- Install MinGW-w64 or TDM-GCC

### Missing Dependencies

If `go mod download` fails:

```bash
# Clear module cache
go clean -modcache

# Re-download
go mod download

# Verify modules
go mod verify
```

## Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Specific package
go test ./internal/service
```

## Installation

### System-wide Installation

**macOS/Linux:**
```bash
sudo cp media-server /usr/local/bin/
sudo chmod +x /usr/local/bin/media-server
```

**Windows (PowerShell as Admin):**
```powershell
Copy-Item media-server.exe C:\Windows\System32\
```

## Development Workflow

1. **Make changes to code**
2. **Run tests:** `go test ./...`
3. **Build:** `go build -o media-server cmd/server/main.go`
4. **Test locally:** `./media-server start`
5. **Commit changes**

## IDE Setup

### VS Code

Install extensions:
- Go (golang.go)
- Go Test Explorer (premparihar.gotestexplorer)

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server",
      "args": ["start"]
    }
  ]
}
```

### GoLand

1. Open server directory
2. Right-click `cmd/server/main.go`
3. Select "Run 'go build main.go'"
4. Edit configuration to add "start" argument

## Performance Tips

### Optimize Binary Size

```bash
# Use upx compression
upx --best --lzma media-server

# Reduce by ~50-70%
```

### Static Linking

```bash
go build -ldflags="-extldflags=-static" -o media-server cmd/server/main.go
```

Good for deployment to systems without libc.
