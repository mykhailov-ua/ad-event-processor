// Package wasmattest loads the C attest WASM module in a bounded wazero sandbox.
//
// Role:
// - Server: dry-run and verify parity with pkg/antifraudtelemetry PoW and runtime probes.
// - Client: same .wasm artifact served from /static/attest.wasm (see internal/track/wasm_attest_loader.js).
//
// ABI (wasm/attest/abi.h):
// - Linear memory export "memory", size AAD_MEM_SIZE (4096).
// - aad_abi_version() -> u32
// - aad_pow_solve(salt_off, difficulty, nonce_start, max_tries) -> nonce or 0xffffffff
// - aad_float_noise_ieee(out_off) -> 0 on success, writes 32-byte SHA-256
// - aad_sha256_one_shot(msg_off, msg_len, out_off) -> 0 on success; bounds-checked hash into linear memory
// - aad_bench_mul(rounds) -> u32 accumulator (deterministic micro-bench)
//
// Invariants:
// - Module must have zero imports (pure compute); load rejects unknown imports.
// - MaxModuleBytes and MaxMemoryPages cap resource use (BPF-analog verifier at load).
// - Not on tracker /track hot path; cold attestation / admin sandbox only.
//
// Verify:
// bash scripts/build/wasm_attest.sh
// bash scripts/ci/static/wasm_attest_gate.sh
// go test ./pkg/wasmattest/ -short -run WasmAttest -count=1
package wasmattest
