BINARY        := gowhisper
APP_BUNDLE    := GoWhisper.app/Contents/MacOS/$(BINARY)
WHISPER_SRC   := third_party/whisper.cpp
WHISPER_BUILD := $(WHISPER_SRC)/build

# Paths for CGo
INCLUDE_PATH  := $(abspath $(WHISPER_SRC)/include):$(abspath $(WHISPER_SRC)/ggml/include)
LIBRARY_PATH  := $(abspath $(WHISPER_BUILD)/src):$(abspath $(WHISPER_BUILD)/ggml/src):$(abspath $(WHISPER_BUILD)/ggml/src/ggml-metal):$(abspath $(WHISPER_BUILD)/ggml/src/ggml-blas)

# macOS Metal/BLAS frameworks required by ggml
EXT_LDFLAGS   := -lggml-metal -lggml-blas -framework Foundation -framework Metal -framework MetalKit -framework Accelerate -framework ApplicationServices -framework CoreGraphics

CGO_ENV       := CGO_ENABLED=1 \
                 C_INCLUDE_PATH="$(INCLUDE_PATH)" \
                 LIBRARY_PATH="$(LIBRARY_PATH)" \
                 GGML_METAL_PATH_RESOURCES="$(abspath $(WHISPER_SRC))"

BUILD_FLAGS   := -buildvcs=false -ldflags "-extldflags '$(EXT_LDFLAGS)'"

.PHONY: all build run test install whisper download-model clean

all: build

## Build whisper.cpp static libraries (run once, or after submodule update)
whisper:
	cmake -S $(WHISPER_SRC) -B $(WHISPER_BUILD) \
		-DCMAKE_BUILD_TYPE=Release \
		-DBUILD_SHARED_LIBS=OFF \
		-DWHISPER_BUILD_TESTS=OFF \
		-DWHISPER_BUILD_EXAMPLES=OFF
	cmake --build $(WHISPER_BUILD) --target whisper ggml -j$$(sysctl -n hw.logicalcpu)

## Compile the Go binary
build:
	@echo "Building $(BINARY)..."
	$(CGO_ENV) go build $(BUILD_FLAGS) -o $(APP_BUNDLE) ./cmd/$(BINARY)/
	@echo "Binary: $(APP_BUNDLE)"

## Build and run the compiled binary
run: build
	$(APP_BUNDLE)

## Run directly with go run (faster for development — no compile step)
dev:
	$(CGO_ENV) go run $(BUILD_FLAGS) ./cmd/gowhisper/

## Record 5s and save /tmp/rectest.wav — pass DEV="name" to pick a device
rectest:
	$(CGO_ENV) go run $(BUILD_FLAGS) ./cmd/rectest/ $(DEV)

## Run tests
test:
	$(CGO_ENV) go test $(BUILD_FLAGS) ./...

## Install app bundle to /Applications
install: build
	@echo "Installing GoWhisper.app to /Applications..."
	cp -r GoWhisper.app /Applications/GoWhisper.app
	@echo "Done."

## Download the configured GGML model (default: small)
download-model:
	@mkdir -p ~/.config/gowhisper/models
	@echo "Downloading ggml-small.bin..."
	curl -L --progress-bar \
		"https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin" \
		-o ~/.config/gowhisper/models/ggml-small.bin
	@echo "Saved to ~/.config/gowhisper/models/ggml-small.bin"

clean:
	rm -f $(APP_BUNDLE)
	go clean
