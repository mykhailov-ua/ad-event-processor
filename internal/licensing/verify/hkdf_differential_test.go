//go:build differential

package verify_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/hkdf"

	"github.com/stretchr/testify/require"
)

func opensslHKDF(t *testing.T, ikm, salt, info []byte, length int) []byte {
	t.Helper()
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not in PATH")
	}
	args := []string{
		"kdf", "-binary",
		"-keylen", fmt.Sprintf("%d", length),
		"-kdfopt", "digest:SHA256",
		"-kdfopt", "hexkey:" + hex.EncodeToString(ikm),
	}
	if len(salt) > 0 {
		args = append(args, "-kdfopt", "hexsalt:"+hex.EncodeToString(salt))
	}
	if len(info) > 0 {
		args = append(args, "-kdfopt", "hexinfo:"+hex.EncodeToString(info))
	}
	args = append(args, "HKDF")
	out, err := exec.Command("openssl", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("openssl hkdf: %v: %s", err, strings.TrimSpace(string(out)))
	}
	require.Len(t, out, length)
	return out
}

func goHKDF(ikm, salt, info []byte, length int) ([]byte, error) {
	reader := hkdf.New(sha256.New, ikm, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, err
	}
	return out, nil
}

func TestHKDF_DifferentialOpenSSL_RFC5869(t *testing.T) {
	path := filepath.Join("testdata", "hkdf_rfc5869.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Fixtures []struct {
			Name    string `json:"name"`
			IKMHex  string `json:"ikm_hex"`
			SaltHex string `json:"salt_hex"`
			InfoHex string `json:"info_hex"`
			Length  int    `json:"length"`
		} `json:"fixtures"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	for _, fx := range doc.Fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			ikm, err := hex.DecodeString(fx.IKMHex)
			require.NoError(t, err)
			var salt, info []byte
			if fx.SaltHex != "" {
				salt, err = hex.DecodeString(fx.SaltHex)
				require.NoError(t, err)
			}
			if fx.InfoHex != "" {
				info, err = hex.DecodeString(fx.InfoHex)
				require.NoError(t, err)
			}
			goOut, err := goHKDF(ikm, salt, info, fx.Length)
			require.NoError(t, err)
			sslOut := opensslHKDF(t, ikm, salt, info, fx.Length)
			require.Equal(t, goOut, sslOut)
		})
	}
}

func TestDeriveMCK_DifferentialOpenSSL(t *testing.T) {
	ikm := []byte("sig-bytes-payload-bytes-hwid-openssl-cross-check")
	deploymentID := "deployment-openssl-diff"
	goOut, err := deriveMCKBytes(ikm, deploymentID)
	require.NoError(t, err)
	sslOut := opensslHKDF(t, ikm, []byte(deploymentID), []byte(MCKInfoLabel()), 32)
	require.Equal(t, goOut[:], sslOut)
}
