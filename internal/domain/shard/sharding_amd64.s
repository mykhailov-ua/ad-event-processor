//go:build amd64

#include "textflag.h"


TEXT ·crc32Castagnoli_asm(SB), NOSPLIT, $0-20
	MOVQ val1+0(FP), AX
	MOVQ val2+8(FP), CX
	MOVL $0xFFFFFFFF, DX
	CRC32Q AX, DX
	CRC32Q CX, DX
	NOTL DX
	MOVL DX, ret+16(FP)
	RET
