package checkout

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCryptoDepositQRSVG_nonEmpty(t *testing.T) {
	svg := cryptoDepositQRSVG("TTestDepositAddressForQRCode123456")
	require.Contains(t, svg, "<svg")
	require.Contains(t, svg, "<rect")
	require.True(t, strings.HasSuffix(strings.TrimSpace(svg), "</svg>"))
}
