package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/hkdf"

	"github.com/stretchr/testify/require"
)

func hkdfSHA256(ikm, salt, info []byte, length int) ([]byte, error) {
	reader := hkdf.New(sha256.New, ikm, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, err
	}
	return out, nil
}

func TestHKDF_RFC5869Vectors(t *testing.T) {
	path := filepath.Join("testdata", "vectors", "hkdf_rfc5869.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Fixtures []struct {
			Name    string `json:"name"`
			IKMHex  string `json:"ikm_hex"`
			SaltHex string `json:"salt_hex"`
			InfoHex string `json:"info_hex"`
			Length  int    `json:"length"`
			OKMHex  string `json:"okm_hex"`
		} `json:"fixtures"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Fixtures)
	for _, fx := range doc.Fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			ikm, err := hex.DecodeString(fx.IKMHex)
			require.NoError(t, err)
			var salt []byte
			if fx.SaltHex != "" {
				salt, err = hex.DecodeString(fx.SaltHex)
				require.NoError(t, err)
			}
			var info []byte
			if fx.InfoHex != "" {
				info, err = hex.DecodeString(fx.InfoHex)
				require.NoError(t, err)
			}
			got, err := hkdfSHA256(ikm, salt, info, fx.Length)
			require.NoError(t, err)
			require.Equal(t, fx.OKMHex, hex.EncodeToString(got))
		})
	}
}

// MCK derivation uses the same HKDF primitive as hkdfSHA256 (deriveMCKBytes).
func TestDeriveMCK_usesHKDFPrimitive(t *testing.T) {
	ikm := []byte("sig-bytes-payload-bytes-hwid")
	salt := []byte("deployment-abc")
	info := []byte(mckInfoLabel)
	got, err := hkdfSHA256(ikm, salt, info, 32)
	require.NoError(t, err)
	want, err := deriveMCKBytes(ikm, "deployment-abc")
	require.NoError(t, err)
	require.Equal(t, want, [32]byte(got))
}
