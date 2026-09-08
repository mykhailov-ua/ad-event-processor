'use strict';

/**
 * Minimal browser loader for wasm/attest C module (wasm32-unknown-unknown, no imports).
 * Lazy-load only on attestation strict tier; not used on /track hot pixel.
 *
 * Verify:
 * go test ./internal/track/ -short -run WasmAttestLoader -count=1
 */
(() => {
  const ABI_VERSION = 1;
  const MEM_SIZE = 4096;
  const SALT_LEN = 16;
  const HASH_LEN = 32;
  const POW_NOT_FOUND = 0xffffffff;
  const SALT_OFF = 0;
  const HASH_OFF = 64;

  let modPromise = null;

  async function loadModule(wasmUrl) {
    if (!modPromise) {
      modPromise = (async () => {
        const resp = await fetch(wasmUrl, { credentials: 'same-origin', cache: 'default' });
        if (!resp.ok) {
          throw new Error('wasm fetch failed');
        }
        const bytes = await resp.arrayBuffer();
        const { instance } = await WebAssembly.instantiate(bytes, {});
        return instance;
      })();
    }
    return modPromise;
  }

  function u32(view, off) {
    return view.getUint32(off, true);
  }

  async function solvePoW(wasmUrl, saltBytes, difficulty, maxTries) {
    const inst = await loadModule(wasmUrl);
    const ver = inst.exports.aad_abi_version();
    if (ver !== ABI_VERSION) {
      return POW_NOT_FOUND;
    }
    const dataOff = inst.exports.aad_data_off();
    const mem = new Uint8Array(inst.exports.memory.buffer);
    if (saltBytes.length !== SALT_LEN) {
      return POW_NOT_FOUND;
    }
    mem.set(saltBytes, dataOff + SALT_OFF);
    const nonce = inst.exports.aad_pow_solve(SALT_OFF, difficulty | 0, 0, maxTries | 0) >>> 0;
    return nonce;
  }

  async function floatNoiseIEEE(wasmUrl) {
    const inst = await loadModule(wasmUrl);
    const dataOff = inst.exports.aad_data_off();
    const mem = new Uint8Array(inst.exports.memory.buffer);
    const hashOff = dataOff + HASH_OFF;
    const status = inst.exports.aad_float_noise_ieee(HASH_OFF);
    if (status !== 0) {
      return '';
    }
    let out = '';
    for (let i = 0; i < HASH_LEN; i += 1) {
      out += mem[hashOff + i].toString(16).padStart(2, '0');
    }
    return out;
  }

  async function benchMul(wasmUrl, rounds) {
    const inst = await loadModule(wasmUrl);
    return inst.exports.aad_bench_mul(rounds | 0) >>> 0;
  }

  globalThis.aedWasmAttest = {
    ABI_VERSION,
    MEM_SIZE,
    loadModule,
    solvePoW,
    floatNoiseIEEE,
    benchMul,
  };
})();
