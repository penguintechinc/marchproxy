# MarchProxy eBPF Build System

LLVM_STRIP ?= llvm-strip
CLANG ?= clang
LLC ?= llc
ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')

# Directories
SRC_DIR := src
BUILD_DIR := build
INCLUDE_DIR := include

# Kernel headers
KERNEL_REL := $(shell uname -r)
KERNEL_SRC := /lib/modules/$(KERNEL_REL)/build

# BPF headers - try different common locations
BPF_HEADERS := $(shell find /usr/include -name "bpf_helpers.h" 2>/dev/null | head -1 | xargs dirname)
ifeq ($(BPF_HEADERS),)
    BPF_HEADERS := /usr/src/linux-headers-$(KERNEL_REL)/tools/lib/bpf
endif
ifeq ($(BPF_HEADERS),)
    BPF_HEADERS := /usr/include/bpf
endif

# Compiler flags
CLANG_BPF_SYS_INCLUDES := $(shell $(CLANG) -v -E - </dev/null 2>&1 \
	| sed -n '/<...> search starts here:/,/End of search list./{ s| \(/.*\)|-I\1|p }')

BPF_CFLAGS := -g -O2 -Wall
BPF_CFLAGS += -target bpf
BPF_CFLAGS += -D__TARGET_ARCH_$(ARCH)
BPF_CFLAGS += -I$(INCLUDE_DIR)
BPF_CFLAGS += -I$(BPF_HEADERS)
BPF_CFLAGS += -I$(KERNEL_SRC)/arch/$(ARCH)/include
BPF_CFLAGS += -I$(KERNEL_SRC)/arch/$(ARCH)/include/generated
BPF_CFLAGS += -I$(KERNEL_SRC)/include
BPF_CFLAGS += -I$(KERNEL_SRC)/arch/$(ARCH)/include/uapi
BPF_CFLAGS += -I$(KERNEL_SRC)/arch/$(ARCH)/include/generated/uapi
BPF_CFLAGS += -I$(KERNEL_SRC)/include/uapi
BPF_CFLAGS += -I$(KERNEL_SRC)/include/generated/uapi
BPF_CFLAGS += $(CLANG_BPF_SYS_INCLUDES)

# Source files
SOURCES := $(wildcard $(SRC_DIR)/*.c $(SRC_DIR)/*.bpf.c)
OBJECTS := $(patsubst $(SRC_DIR)/%.c,$(BUILD_DIR)/%.o,$(filter %.c,$(SOURCES)))
OBJECTS += $(patsubst $(SRC_DIR)/%.bpf.c,$(BUILD_DIR)/%.o,$(filter %.bpf.c,$(SOURCES)))

.PHONY: all clean setup check-deps

all: setup $(OBJECTS)

setup:
	@mkdir -p $(BUILD_DIR)

check-deps:
	@echo "Checking eBPF build dependencies..."
	@command -v $(CLANG) >/dev/null 2>&1 || { echo "Error: clang not found"; exit 1; }
	@command -v $(LLVM_STRIP) >/dev/null 2>&1 || { echo "Error: llvm-strip not found"; exit 1; }
	@test -d $(KERNEL_SRC) || { echo "Error: Kernel headers not found at $(KERNEL_SRC)"; exit 1; }
	@echo "Build environment OK"
	@echo "Kernel: $(KERNEL_REL)"
	@echo "Architecture: $(ARCH)"
	@echo "BPF Headers: $(BPF_HEADERS)"

$(BUILD_DIR)/%.o: $(SRC_DIR)/%.c
	@echo "Compiling eBPF program: $<"
	$(CLANG) $(BPF_CFLAGS) -c $< -o $@
	$(LLVM_STRIP) -g $@

clean:
	rm -rf $(BUILD_DIR)

install: all
	@echo "Installing eBPF programs to ../proxy/ebpf/"
	@mkdir -p ../proxy/ebpf
	cp $(BUILD_DIR)/*.o ../proxy/ebpf/

# Development targets
dev-info:
	@echo "=== eBPF Development Information ==="
	@echo "Sources: $(SOURCES)"
	@echo "Objects: $(OBJECTS)"
	@echo "Compiler: $(CLANG)"
	@echo "Flags: $(BPF_CFLAGS)"
	@echo ""

# Test compilation with verbose output
test-compile: setup
	@echo "Test compiling with verbose output..."
	$(CLANG) $(BPF_CFLAGS) -v -c $(SRC_DIR)/marchproxy.c -o $(BUILD_DIR)/test.o 2>&1 || true

help:
	@echo "MarchProxy eBPF Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all        - Build all eBPF programs"
	@echo "  clean      - Clean build artifacts"  
	@echo "  install    - Install programs to proxy directory"
	@echo "  check-deps - Verify build dependencies"
	@echo "  dev-info   - Show development information"
	@echo "  test-compile - Test compilation with verbose output"
	@echo "  help       - Show this help"