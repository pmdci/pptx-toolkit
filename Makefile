# Build variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")
LDFLAGS_COMMON := -s -w -X main.Version=$(VERSION)
LDFLAGS_RELEASE := $(LDFLAGS_COMMON) -extldflags=-Wl,--strip-all
BUILD_FLAGS := -trimpath
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: build build-release cross-compile clean test install dev

# Default build with optimization
build:
	mkdir -p bin
	go build -ldflags="$(LDFLAGS_COMMON)" $(BUILD_FLAGS) -o bin/pptx-toolkit ./cmd/pptx-toolkit
	@echo "🚀 Built pptx-toolkit $(VERSION)"

# Release build with maximum optimization
build-release:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS_RELEASE)" $(BUILD_FLAGS) -o bin/pptx-toolkit ./cmd/pptx-toolkit
	@if command -v upx >/dev/null 2>&1; then \
		echo "UPX found, compressing binary..."; \
		if [ "$(shell uname)" = "Darwin" ]; then \
			echo "UPX compression for macOS is officially unsupported until further notice. Skipping..."; \
		else \
			upx --best bin/pptx-toolkit; \
		fi; \
	else \
		echo "UPX not found, skipping compression (binary size: $(shell du -h bin/pptx-toolkit | cut -f1))"; \
	fi
	@echo "🚀 Built pptx-toolkit $(VERSION)"

# Cross-compile for multiple platforms
cross-compile:
	mkdir -p bin
	@echo "Building for multiple platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 go build -ldflags="$(LDFLAGS_COMMON)" $(BUILD_FLAGS) \
		-o bin/pptx-toolkit-$${platform%/*}-$${platform#*/} ./cmd/pptx-toolkit; \
	done
	@if command -v upx >/dev/null 2>&1; then \
		echo "UPX found, compressing cross-compiled binaries..."; \
		echo "UPX compression for macOS is officially unsupported until further notice. Skipping Darwin binaries..."; \
		echo "UPX does not yet support Windows ARM64 PE format. Skipping windows-arm64 binary..."; \
		upx --best bin/pptx-toolkit-linux-* bin/pptx-toolkit-windows-amd64 2>/dev/null || true; \
		echo "Cross-compilation and compression complete."; \
	else \
		echo "UPX not found, skipping compression."; \
		echo "Cross-compilation complete. Binaries in bin/"; \
	fi
	@echo "🚀 Built pptx-toolkit $(VERSION) for $(PLATFORMS)"

clean:
	rm -rf bin/

test:
	go test ./...

install: build
	@mkdir -p $(HOME)/.local/bin
	cp bin/pptx-toolkit $(HOME)/.local/bin/
	@echo "✓ Installed to $(HOME)/.local/bin/pptx-toolkit"

dev: build
	./bin/pptx-toolkit $(ARGS)
