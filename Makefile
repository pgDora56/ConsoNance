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

# Build for Mac ARM (Apple Silicon)
build-mac-arm:
	go build -o $(BINARY_NAME)-mac-arm main.go

# Build for Mac Intel (x86_64)
build-mac-intel:
	go build -o $(BINARY_NAME)-mac-intel main.go

# Deploy the documentation
deploy-docs:
	rsync -avz --delete docs/ nance:/deploy/consonance/docs/