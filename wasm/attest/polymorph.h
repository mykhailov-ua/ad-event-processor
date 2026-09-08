#ifndef AAD_POLY_H
#define AAD_POLY_H

#include <stdint.h>

#ifndef AAD_POLY_SEED
#define AAD_POLY_SEED 0u
#endif

#ifndef AAD_POLY_JUNK_LEN
#define AAD_POLY_JUNK_LEN 16u
#endif

#ifndef AAD_POLY_EXTRA_FN
#define AAD_POLY_EXTRA_FN 0u
#endif

extern const uint8_t aad_poly_junk[AAD_POLY_JUNK_LEN];

uint32_t aad_polymorph_touch(void);

#endif
