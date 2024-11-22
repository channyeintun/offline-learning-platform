APP_NAME := app
SOURCE := main.go

all: build-mac build-windows

build-mac:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o $(APP_NAME)-mac $(SOURCE)

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(APP_NAME)-windows.exe $(SOURCE)

clean:
	@echo "Cleaning up build artifacts..."
	rm -f $(APP_NAME)-mac $(APP_NAME)-windows.exe

.PHONY: all build-mac build-windows clean