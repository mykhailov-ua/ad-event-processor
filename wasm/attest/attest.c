#include "abi.h"
#include "polymorph.h"
#include "sha256.h"
#include <stddef.h>

uint8_t aad_mem[AAD_MEM_SIZE];

static int pow_hash_ok(const uint8_t *salt, uint32_t nonce, uint32_t difficulty) {
    uint8_t buf[20];
    uint8_t digest[32];
    uint32_t i;
    aad_sha256_ctx ctx;

    for (i = 0; i < AAD_SALT_LEN; ++i) {
        buf[i] = salt[i];
    }
    buf[16] = (uint8_t)((nonce >> 24) & 0xffu);
    buf[17] = (uint8_t)((nonce >> 16) & 0xffu);
    buf[18] = (uint8_t)((nonce >> 8) & 0xffu);
    buf[19] = (uint8_t)(nonce & 0xffu);

    aad_sha256_init(&ctx);
    aad_sha256_update(&ctx, buf, 20u);
    aad_sha256_final(&ctx, digest);

    for (i = 0; i < difficulty; ++i) {
        if (digest[i] != 0u) {
            return 0;
        }
    }
    return 1;
}

__attribute__((export_name("aad_abi_version"))) uint32_t aad_abi_version(void) { return AAD_ABI_VERSION; }

__attribute__((export_name("aad_poly_seed"))) uint32_t aad_poly_seed(void) { return AAD_POLY_SEED; }

__attribute__((export_name("aad_poly_tag"))) uint32_t aad_poly_tag(void) {
	return (uint32_t)aad_poly_junk[0] | ((uint32_t)aad_poly_junk[1] << 8);
}

__attribute__((export_name("aad_data_off"))) uint32_t aad_data_off(void) {
    return (uint32_t)(uintptr_t)aad_mem;
}

__attribute__((export_name("aad_pow_solve")))
uint32_t aad_pow_solve(uint32_t salt_off, uint32_t difficulty, uint32_t nonce_start, uint32_t max_tries) {
    uint32_t i;
    const uint8_t *salt;

    if (difficulty == 0u || difficulty > AAD_MAX_POW_DIFFICULTY) {
        return AAD_POW_NOT_FOUND;
    }
    if (salt_off + AAD_SALT_LEN > AAD_MEM_SIZE) {
        return AAD_POW_NOT_FOUND;
    }
    salt = aad_mem + salt_off;
    for (i = 0; i < max_tries; ++i) {
        uint32_t nonce = nonce_start + i;
        if (pow_hash_ok(salt, nonce, difficulty)) {
            return nonce;
        }
    }
    return AAD_POW_NOT_FOUND;
}

__attribute__((export_name("aad_float_noise_ieee")))
uint32_t aad_float_noise_ieee(uint32_t out_off) {
    static const uint8_t ieee_lit[AAD_FLOAT_NOISE_IEEE_LEN] = {
        '0', '.', '3', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '4',
    };
    aad_sha256_ctx ctx;

    if (out_off + AAD_HASH_LEN > AAD_MEM_SIZE) {
        return 1u;
    }
    aad_sha256_init(&ctx);
    aad_sha256_update(&ctx, ieee_lit, AAD_FLOAT_NOISE_IEEE_LEN);
    aad_sha256_final(&ctx, aad_mem + out_off);
    return 0u;
}

__attribute__((export_name("aad_sha256_one_shot")))
uint32_t aad_sha256_one_shot(uint32_t msg_off, uint32_t msg_len, uint32_t out_off) {
    if (msg_len > AAD_MEM_SIZE || out_off + AAD_HASH_LEN > AAD_MEM_SIZE) {
        return 1u;
    }
    if (msg_off > AAD_MEM_SIZE - msg_len) {
        return 1u;
    }
    aad_sha256_digest(aad_mem + msg_off, msg_len, aad_mem + out_off);
    return 0u;
}

__attribute__((export_name("aad_bench_mul")))
uint32_t aad_bench_mul(uint32_t rounds) {
    uint32_t i;
    uint32_t acc = 0x9e3779b9u;
	if (rounds > 2000000u) {
		rounds = 2000000u;
	}
	for (i = 0; i < rounds; ++i) {
        acc = (acc * 1664525u) + 1013904223u;
    }
    return acc;
}
