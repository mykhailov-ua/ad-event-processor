// Sync SHA-1 for UUID v5 derivation in browser and Node test runners (no async crypto).

function rotl(value: number, shift: number): number {
  return (value << shift) | (value >>> (32 - shift));
}

export function sha1(message: Uint8Array): Uint8Array {
  const words: number[] = [];
  for (let index = 0; index < message.length; index += 1) {
    words[index >> 2] |= message[index] << (24 - (index % 4) * 8);
  }
  words[message.length >> 2] |= 0x80 << (24 - (message.length % 4) * 8);
  const tail = (((message.length + 8) >> 6) + 1) * 16;
  while (words.length < tail) {
    words.push(0);
  }
  words[tail - 1] = message.length * 8;

  let a = 0x67452301;
  let b = 0xefcdab89;
  let c = 0x98badcfe;
  let d = 0x10325476;
  let e = 0xc3d2e1f0;

  for (let offset = 0; offset < words.length; offset += 16) {
    const w = new Array<number>(80);
    for (let index = 0; index < 16; index += 1) {
      w[index] = words[offset + index] | 0;
    }
    for (let index = 16; index < 80; index += 1) {
      w[index] = rotl(w[index - 3] ^ w[index - 8] ^ w[index - 14] ^ w[index - 16], 1);
    }

    let aa = a;
    let bb = b;
    let cc = c;
    let dd = d;
    let ee = e;

    for (let index = 0; index < 80; index += 1) {
      let f: number;
      let k: number;
      if (index < 20) {
        f = (bb & cc) | (~bb & dd);
        k = 0x5a827999;
      } else if (index < 40) {
        f = bb ^ cc ^ dd;
        k = 0x6ed9eba1;
      } else if (index < 60) {
        f = (bb & cc) | (bb & dd) | (cc & dd);
        k = 0x8f1bbcdc;
      } else {
        f = bb ^ cc ^ dd;
        k = 0xca62c1d6;
      }
      const temp = (rotl(aa, 5) + f + ee + k + w[index]) | 0;
      ee = dd;
      dd = cc;
      cc = rotl(bb, 30);
      bb = aa;
      aa = temp;
    }

    a = (a + aa) | 0;
    b = (b + bb) | 0;
    c = (c + cc) | 0;
    d = (d + dd) | 0;
    e = (e + ee) | 0;
  }

  const out = new Uint8Array(20);
  const state = [a, b, c, d, e];
  for (let index = 0; index < state.length; index += 1) {
    out[index * 4] = (state[index] >>> 24) & 0xff;
    out[index * 4 + 1] = (state[index] >>> 16) & 0xff;
    out[index * 4 + 2] = (state[index] >>> 8) & 0xff;
    out[index * 4 + 3] = state[index] & 0xff;
  }
  return out;
}
