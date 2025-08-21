# Build variables
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS_COMMON := -s -w -X main.Version=$(VERSION)
LDFLAGS_RELEASE := $(LDFLAGS_COMMON) -extldflags=-Wl,--strip-all
BUILD_FLAGS := -trimpath
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build build-release cross-compile clean test install dev

# Default build with optimization (suitable for development and general use)
build:
	mkdir -p bin
	go build -ldflags="$(LDFLAGS_COMMON)" $(BUILD_FLAGS) -o bin/pathuni ./cmd/pathuni

# Release build with maximum optimization and static linking
build-release:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS_RELEASE)" $(BUILD_FLAGS) -o bin/pathuni ./cmd/pathuni
	@if command -v upx >/dev/null 2>&1; then \
		echo "UPX found, compressing binary..."; \
		if [ "$$(uname)" = "Darwin" ]; then \
			upx --force-macos --best bin/pathuni; \
		else \
			upx --best bin/pathuni; \
		fi; \
	else \
		echo "UPX not found, skipping compression (binary size: $$(du -h bin/pathuni | cut -f1))"; \
	fi

# Cross-compile for multiple platforms
cross-compile:
	mkdir -p bin
	@echo "Building for multiple platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 go build -ldflags="$(LDFLAGS_COMMON)" $(BUILD_FLAGS) \
		-o bin/pathuni-$${platform%/*}-$${platform#*/} ./cmd/pathuni; \
	done
	@if command -v upx >/dev/null 2>&1; then \
		echo "UPX found, compressing cross-compiled binaries..."; \
		upx --force-macos --best bin/pathuni-darwin-* 2>/dev/null || true; \
		upx --best bin/pathuni-linux-* 2>/dev/null || true; \
		echo "Cross-compilation and compression complete."; \
	else \
		echo "UPX not found, skipping compression."; \
		echo "Cross-compilation complete. Binaries in bin/"; \
	fi

clean:
	rm -rf bin/

test:
	go test ./...

install: build
	@mkdir -p $(HOME)/.local/bin
	cp bin/pathuni $(HOME)/.local/bin/

dev: build
	./bin/pathuni --eval