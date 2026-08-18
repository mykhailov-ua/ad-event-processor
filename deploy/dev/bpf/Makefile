CLANG ?= clang
ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
BPF_CFLAGS := -O2 -g -Wall -Werror -target bpf -D__TARGET_ARCH_$(ARCH)
BPF_INCLUDES := -I. -I/usr/include -I/usr/include/$(shell uname -m)-linux-gnu -I/usr/local/include

OUT_DIR := .
OBJ := $(OUT_DIR)/loadtest_probe.o

.PHONY: all clean disasm

all: $(OBJ)

$(OBJ): loadtest_probe.bpf.c espx_probe.h
	$(CLANG) $(BPF_CFLAGS) $(BPF_INCLUDES) -c loadtest_probe.bpf.c -o $@

clean:
	rm -f $(OBJ)

disasm: $(OBJ)
	llvm-objdump -d -S $(OBJ) > loadtest_probe.disasm.txt
