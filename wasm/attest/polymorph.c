#include "polymorph.h"

__attribute__((used)) static volatile const uint8_t *aad_poly_ro_anchor = aad_poly_junk;

uint32_t aad_polymorph_touch(void) {
	uint32_t acc = AAD_POLY_SEED;
	uint32_t i;
	for (i = 0; i < AAD_POLY_JUNK_LEN; ++i) {
		acc ^= aad_poly_junk[i];
		acc = (acc * 1664525u) + 1013904223u;
	}
#if AAD_POLY_EXTRA_FN >= 1
	acc ^= (acc >> 13) | (acc << 19);
#endif
#if AAD_POLY_EXTRA_FN >= 2
	acc += AAD_POLY_SEED ^ 0x85ebca6bu;
#endif
#if AAD_POLY_EXTRA_FN >= 3
	acc ^= (uint32_t)AAD_POLY_JUNK_LEN * 0x9e3779b9u;
#endif
	return acc;
}
