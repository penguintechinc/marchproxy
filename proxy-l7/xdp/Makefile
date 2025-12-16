# Makefile for building Envoy XDP program

CLANG ?= clang
LLC ?= llc
OPT ?= opt
LLVM_DIS ?= llvm-dis

KERNEL_SRC ?= /lib/modules/$(shell uname -r)/build
LIBBPF_DIR ?= /usr/include

# Try to find kernel headers in multiple locations
ifeq ($(wildcard $(KERNEL_SRC)/include),)
    # Fallback to installed headers
    KERNEL_HEADERS := $(shell find /usr/src -name "linux-headers-*" -type d 2>/dev/null | head -n 1)
    ifneq ($(KERNEL_HEADERS),)
        KERNEL_SRC := $(KERNEL_HEADERS)
    else
        KERNEL_SRC := /usr
    endif
endif

CFLAGS := -O2 -g -Wall -Werror
CFLAGS += -I$(LIBBPF_DIR)
CFLAGS += -I/usr/include
CFLAGS += -I/usr/include/x86_64-linux-gnu
# Only add kernel-specific includes if they exist
ifneq ($(wildcard $(KERNEL_SRC)/include),)
CFLAGS += -I$(KERNEL_SRC)/include
CFLAGS += -I$(KERNEL_SRC)/include/uapi
CFLAGS += -I$(KERNEL_SRC)/include/generated/uapi
CFLAGS += -I$(KERNEL_SRC)/arch/x86/include
CFLAGS += -I$(KERNEL_SRC)/arch/x86/include/uapi
CFLAGS += -I$(KERNEL_SRC)/arch/x86/include/generated
CFLAGS += -I$(KERNEL_SRC)/arch/x86/include/generated/uapi
endif
CFLAGS += -D__KERNEL__ -D__BPF_TRACING__
CFLAGS += -Wno-unused-value -Wno-pointer-sign
CFLAGS += -Wno-compare-distinct-pointer-types
CFLAGS += -Wno-gnu-variable-sized-type-not-at-end
CFLAGS += -Wno-address-of-packed-member
CFLAGS += -Wno-tautological-compare
CFLAGS += -Wno-unknown-warning-option

TARGET := envoy_xdp.o

all: $(TARGET)

$(TARGET): envoy_xdp.c
	$(CLANG) -S \
		-target bpf \
		$(CFLAGS) \
		-emit-llvm -c -o ${@:.o=.ll} $<
	$(LLC) -march=bpf -filetype=obj -o $@ ${@:.o=.ll}

clean:
	rm -f *.o *.ll

install: $(TARGET)
	@echo "XDP program built successfully: $(TARGET)"

.PHONY: all clean install
