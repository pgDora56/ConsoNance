.PHONY: build run clean deps list-devices

# Binary name
BINARY_NAME=consonance

# Build the application
build:
	go build -o $(BINARY_NAME) main.go

# Run the application
run: build
	./$(BINARY_NAME)

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe main.go

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 main.go

# Build for Mac
build-mac:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 main.go