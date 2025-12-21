.PHONY: build run clean deps list-devices build-win build-windows build-linux build-mac

# Binary name
BINARY_NAME=consonance

# Build the application
build:
	go build -o $(BINARY_NAME) main.go

# Run the application
run: build
	./$(BINARY_NAME)

# Build for Windows (run in PowerShell)
# Direct command: $env:PATH += ";C:\msys64\mingw64\bin"; $env:CGO_ENABLED=1; go build -o consonance-win.exe
build-win:
	powershell -Command "$$env:PATH += ';C:\msys64\mingw64\bin'; $$env:CGO_ENABLED=1; go build -o consonance-win.exe"

# Build for Windows (cross-compile)
build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe main.go

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 main.go

# Build for Mac
build-mac:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 main.go