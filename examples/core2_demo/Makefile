# Makefile for core2_demo
# This demo shows how to trigger core dumps with CGO

# Compiler and flags
CC = gcc
CFLAGS = -fPIC -shared -Wall
GO = go

# Targets
LIBADD = libadd.so
DEMO = demo

# Default target
all: $(DEMO)

# Compile C library
$(LIBADD): add.c
	$(CC) $(CFLAGS) -o $@ $<

# Compile Go program
$(DEMO): main.go $(LIBADD)
	$(GO) build -o $@ $<

# Run the demo
run: $(DEMO)
	@echo "Running core2_demo..."
	@echo "Setting up core dump environment..."
	ulimit -c unlimited
	sudo sysctl -w kernel.core_pattern=demo.core
	LD_LIBRARY_PATH=.:$$LD_LIBRARY_PATH GOTRACEBACK=crash ./$(DEMO)

# Clean build artifacts
clean:
	rm -f $(LIBADD) $(DEMO) demo.core*

# Show help
help:
	@echo "Available targets:"
	@echo "  all     - Build the demo (default)"
	@echo "  run     - Build and run the demo"
	@echo "  clean   - Remove build artifacts"
	@echo "  help    - Show this help message"

.PHONY: all run clean help
