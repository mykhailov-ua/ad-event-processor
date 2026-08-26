package proxyupstream_test

import (
	"context"
	"testing"

	"ad-event-processor/pkg/proxyupstream"

	"github.com/stretchr/testify/require"
)

func TestValidateURL_blocksPrivateLiteral(t *testing.T) {
	err := proxyupstream.ValidateURL(context.Background(), "https://10.0.0.1/lp", false)
	require.ErrorIs(t, err, proxyupstream.ErrPrivateUpstream)
}

func TestValidateURL_requiresHTTPS(t *testing.T) {
	err := proxyupstream.ValidateURL(context.Background(), "http://example.com/lp", false)
	require.ErrorIs(t, err, proxyupstream.ErrInvalidScheme)
}

func TestValidateURL_allowsHTTPLab(t *testing.T) {
	err := proxyupstream.ValidateURL(context.Background(), "http://93.184.216.34/", true)
	require.NoError(t, err)
}

func TestValidateDeliveryPair_proxyRequiresURL(t *testing.T) {
	err := proxyupstream.ValidateDeliveryPair(context.Background(), proxyupstream.ClickDeliveryProxy, "", false)
	require.ErrorIs(t, err, proxyupstream.ErrEmptyURL)
}

func TestValidateDeliveryPair_redirectOK(t *testing.T) {
	err := proxyupstream.ValidateDeliveryPair(context.Background(), proxyupstream.ClickDeliveryRedirect, "", false)
	require.NoError(t, err)
}
