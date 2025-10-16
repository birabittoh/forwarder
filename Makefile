.PHONY: all build-tdlib install-tdlib build run docker-build clean help

# Default target
all: build

# Variables
TDLIB_DIR := tdlib
TDLIB_BUILD_DIR := $(TDLIB_DIR)/build
NPROC := $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
CGO_LDFLAGS := -L/usr/local/lib -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdmtproto -ltdactor -ltdapi -ltddb -ltdsqlite -ltdnet -ltdutils -ltde2e -lstdc++ -lssl -lcrypto -ldl -lz -lm -lpthread
CGO_CFLAGS := -I/usr/local/include

# Build tdlib from source
build-tdlib:
	@echo "Building tdlib..."
	@mkdir -p $(TDLIB_BUILD_DIR)
	@cd $(TDLIB_BUILD_DIR) && \
		cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local .. && \
		cmake --build . -j$(NPROC)
	@echo "tdlib built successfully!"

# Install tdlib to /usr/local
install-tdlib: build-tdlib
	@echo "Installing tdlib..."
	@cd $(TDLIB_BUILD_DIR) && sudo cmake --install .
	@sudo ldconfig 2>/dev/null || true
	@echo "tdlib installed successfully!"

# Build forwarder
build:
	@echo "Building forwarder..."
	@CGO_ENABLED=1 \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CGO_CFLAGS="$(CGO_CFLAGS)" \
		go build -o forwarder .
	@echo "forwarder built successfully!"

# Run forwarder
run: build
	@echo "Running forwarder..."
	@CGO_LDFLAGS="$(CGO_LDFLAGS)" ./forwarder

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker compose build
	@echo "Docker image built successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f forwarder
	@rm -rf $(TDLIB_BUILD_DIR)
	@go clean
	@echo "Clean complete!"

# Show help
help:
	@echo "Available targets:"
	@echo "  all (default)    - Build forwarder"
	@echo "  build-tdlib      - Build tdlib from source"
	@echo "  install-tdlib    - Build and install tdlib to /usr/local"
	@echo "  build            - Build forwarder binary"
	@echo "  run              - Build and run forwarder"
	@echo "  docker-build     - Build Docker image using docker compose"
	@echo "  clean            - Remove build artifacts"
	@echo "  help             - Show this help message"