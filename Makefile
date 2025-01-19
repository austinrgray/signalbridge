MICROSERVICES := cartographer emitter mission_control receiver
BIN_DIR := bin
CMD_DIR := cmd
NATS_COMMAND := nats-server --jetstream

build-%:
	@echo "Building $*..."
	@go build -o $(BIN_DIR)/$* $(CMD_DIR)/$*/main.go

# Run each microservice
run-%:
	@echo "Running $*..."
	@$(BIN_DIR)/$* &

# Build all microservices
build-all:
	@for service in $(MICROSERVICES); do \
		$(MAKE) build-$$service; \
	done

# Run all microservices
run-all:
	@for service in $(MICROSERVICES); do \
		$(MAKE) run-$$service; \
	done

# Start the NATS server
start-nats:
	@echo "Starting NATS server..."
	@$(NATS_COMMAND) &

# Stop all processes (Windows-specific)
stop-all:
	@echo "Stopping all microservices and NATS server..."
	@taskkill /IM nats-server.exe /F >nul 2>&1 || echo "NATS server not running."
	@for service in $(MICROSERVICES); do \
		taskkill /IM $$service.exe /F >nul 2>&1 || echo "$$service not running."; \
	done

# Build, run, and start everything
start-all: build-all start-nats run-all

# Clean up binaries
clean:
	@echo "Cleaning up binaries..."
	@if exist $(BIN_DIR) rmdir /S /Q $(BIN_DIR)
	@mkdir $(BIN_DIR)

.PHONY: build-all run-all start-nats stop-all start-all clean