#ifndef AAD_SHA256_H
#define AAD_SHA256_H

#include <stdint.h>

typedef struct {
    uint32_t state[8];
    uint64_t bitlen;
    uint8_t data[64];
    uint32_t datalen;
} aad_sha256_ctx;

void aad_sha256_init(aad_sha256_ctx *ctx);
void aad_sha256_update(aad_sha256_ctx *ctx, const uint8_t *data, uint32_t len);
void aad_sha256_final(aad_sha256_ctx *ctx, uint8_t out[32]);
void aad_sha256_digest(const uint8_t *msg, uint32_t len, uint8_t out[32]);

#endif
