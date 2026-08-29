//go:build amd64

#include "textflag.h"

TEXT ·hwidRawOpenRead(SB), NOSPLIT, $0-40
	MOVQ $-100, DI
	MOVQ pathPtr+0(FP), SI
	MOVQ pathLen+8(FP), DX
	MOVQ $0, R10
	MOVQ $257, AX
	SYSCALL
	CMPQ AX, $0
	JL open_fail
	MOVQ AX, R9

	MOVQ R9, DI
	MOVQ bufPtr+16(FP), SI
	MOVQ bufLen+24(FP), DX
	MOVQ $0, AX
	SYSCALL
	MOVQ AX, R11

	MOVQ R9, DI
	MOVQ $3, AX
	SYSCALL

	CMPQ R11, $0
	JL read_fail
	MOVQ R11, ret+32(FP)
	RET

open_fail:
	NEGQ AX
	MOVQ AX, ret+32(FP)
	RET

read_fail:
	MOVQ R9, DI
	MOVQ $3, AX
	SYSCALL
	NEGQ R11
	MOVQ R11, ret+32(FP)
	RET
