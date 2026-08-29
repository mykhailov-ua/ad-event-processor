package filter

import (
	"ad-event-processor/internal/filter/netintel"
)

type (
	GeoProvider           = netintel.GeoProvider
	MaxMindProvider       = netintel.MaxMindProvider
	MockGeoProvider       = netintel.MockGeoProvider
	DCASNTable            = netintel.DCASNTable
	DCASNSnapshot         = netintel.DCASNSnapshot
	CIDRTable             = netintel.CIDRTable
	CIDRSnapshot          = netintel.CIDRSnapshot
	CIDRBuilder           = netintel.CIDRBuilder
	CIDRNode              = netintel.CIDRNode
	ProxyVPNTable         = netintel.ProxyVPNTable
	ProxyVPNSnapshot      = netintel.ProxyVPNSnapshot
	ProxyVPNBuilder       = netintel.ProxyVPNBuilder
	ResidentialIntelTable = netintel.ResidentialIntelTable
	ResidentialProxyRing  = netintel.ResidentialProxyRing
	ResidentialProxyRow   = netintel.ResidentialProxyRow
	ModeratorIPTable      = netintel.ModeratorIPTable
	MobileCarrierASNTable = netintel.MobileCarrierASNTable
	GeoIPUpdaterConfig    = netintel.GeoIPUpdaterConfig
	GeoIPUpdater          = netintel.GeoIPUpdater
	GeoIPWatcher          = netintel.GeoIPWatcher
)

var (
	EnsureIngestGeo                      = netintel.EnsureIngestGeo
	AcceptLangGeoMismatch                = netintel.AcceptLangGeoMismatch
	TlsFingerprintImpersonating          = netintel.TlsFingerprintImpersonating
	NewMaxMindProvider                   = netintel.NewMaxMindProvider
	NewDCASNTable                        = netintel.NewDCASNTable
	NewCIDRTable                         = netintel.NewCIDRTable
	NewProxyVPNTable                     = netintel.NewProxyVPNTable
	NewResidentialIntelTable             = netintel.NewResidentialIntelTable
	NewResidentialProxyRing              = netintel.NewResidentialProxyRing
	NewModeratorIPTable                  = netintel.NewModeratorIPTable
	NewMobileCarrierASNTable             = netintel.NewMobileCarrierASNTable
	CIDRFeedNames                        = netintel.CIDRFeedNames
	BuildCIDRTableFromPrefixes           = netintel.BuildCIDRTableFromPrefixes
	BuildDCASNSnapshot                   = netintel.BuildDCASNSnapshot
	ParseASNLine                         = netintel.ParseASNLine
	NewGeoIPWatcher                      = netintel.NewGeoIPWatcher
	NewGeoIPUpdater                      = netintel.NewGeoIPUpdater
	NewDCASNFeedLoader                   = netintel.NewDCASNFeedLoader
	ParseMobileCarrierASNs               = netintel.ParseMobileCarrierASNs
	ResidentialProxyPolicyFromEnv        = netintel.ResidentialProxyPolicyFromEnv
	NewCIDRFeedLoader                    = netintel.NewCIDRFeedLoader
	NewProxyVPNFeedLoader                = netintel.NewProxyVPNFeedLoader
	NewModeratorIntelFeedLoader          = netintel.NewModeratorIntelFeedLoader
	DefaultResidentialProxyPolicyForTest = netintel.DefaultResidentialProxyPolicyForTest
	ResidentialProxySignalForTest        = netintel.ResidentialProxySignalForTest
)

const (
	CIDRFeedCount              = netintel.CIDRFeedCount
	CIDRNoIndex                = netintel.CIDRNoIndex
	CIDRFeedAWS                = netintel.CIDRFeedAWS
	CIDRFeedGCP                = netintel.CIDRFeedGCP
	CIDRFeedAzure              = netintel.CIDRFeedAzure
	CIDRFeedTor                = netintel.CIDRFeedTor
	CIDRFeedOther              = netintel.CIDRFeedOther
	ConnTimingRTTBit           = netintel.ConnTimingRTTBit
	ConnTimingTTFBBit          = netintel.ConnTimingTTFBBit
	ModeratorIntelFeedFileName = netintel.ModeratorIntelFeedFileName
	ModeratorIntelSigFileName  = netintel.ModeratorIntelSigFileName
	ProxyVPNConnISP            = netintel.ProxyVPNConnISP
	ProxyVPNConnHosting        = netintel.ProxyVPNConnHosting
	ProxyVPNConnVPN            = netintel.ProxyVPNConnVPN
	ProxyVPNConnMobile         = netintel.ProxyVPNConnMobile
)
