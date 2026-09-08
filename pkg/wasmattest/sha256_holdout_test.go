package wasmattest

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWasmAttest_sha256NISTVectors_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	ctx := t.Context()
	sb, err := LoadSandbox(ctx, wasm, DefaultLimits())
	require.NoError(t, err)
	defer sb.Close(ctx)

	cases := []struct {
		name string
		msg  []byte
	}{
		{name: "empty", msg: nil},
		{name: "abc", msg: []byte("abc")},
		{
			name: "56_byte_nist_448bit",
			msg:  []byte("abcdbcde" + "fghijklm" + "ghijklmn" + "hijklmno" + "ijklmnop" + "jklmnopq" + "klmnopqr"),
		},
		{name: "64_byte_exact_block", msg: bytesRepeat('a', 64)},
		{
			name: "80_byte_two_block",
			msg: func() []byte {
				m := bytesRepeat('a', 64)
				return append(m, 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i')
			}(),
		},
	}
	for _, tc := range cases {
		var msg []byte
		if tc.msg != nil {
			msg = tc.msg
		}
		got, err := sb.SHA256OneShot(ctx, msg)
		require.NoError(t, err, tc.name)
		want := sha256.Sum256(msg)
		require.Equal(t, hex.EncodeToString(want[:]), hex.EncodeToString(got), tc.name)
	}
}

func TestWasmAttest_sha256FloatNoise_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	ctx := t.Context()
	sb, err := LoadSandbox(ctx, wasm, DefaultLimits())
	require.NoError(t, err)
	defer sb.Close(ctx)

	got, err := sb.FloatNoiseIEEE(ctx)
	require.NoError(t, err)
	want := sha256.Sum256([]byte("0.30000000000000004"))
	require.Equal(t, hex.EncodeToString(want[:]), hex.EncodeToString(got))
}

func TestWasmAttest_powUsesCanonicalSHA256_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	ctx := t.Context()
	sb, err := LoadSandbox(ctx, wasm, DefaultLimits())
	require.NoError(t, err)
	defer sb.Close(ctx)

	salt := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	var buf [20]byte
	copy(buf[:16], salt)

	for nonce := uint32(0); nonce < 100_000; nonce++ {
		buf[16] = byte(nonce >> 24)
		buf[17] = byte(nonce >> 16)
		buf[18] = byte(nonce >> 8)
		buf[19] = byte(nonce)
		sum := sha256.Sum256(buf[:])
		if sum[0] == 0 && sum[1] == 0 {
			got, err := sb.PoWSolve(ctx, salt, 2, nonce+1)
			require.NoError(t, err)
			require.Equal(t, nonce, got)
			return
		}
	}
	t.Fatal("no matching nonce in search window")
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
