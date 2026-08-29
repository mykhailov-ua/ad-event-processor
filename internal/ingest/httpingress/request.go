package httpingress

const HeaderOrderMax = 16

type Request struct {
	Method                []byte
	Path                  []byte
	ContentType           []byte
	ClientIP              []byte
	UserAgent             []byte
	Accept                []byte
	AcceptEncoding        []byte
	TLSHash               []byte
	TLSJA3                []byte
	TLSJA4                []byte
	SecCHUA               []byte
	SecCHUAPlatform       []byte
	SecCHUAMobile         []byte
	SecFetchSite          []byte
	SecFetchMode          []byte
	SecFetchDest          []byte
	TLSALPN               []byte
	SecFetchPresent       uint8
	H2WireFlags           uint8
	H2SettingsCRC         uint32
	H2EnablePush          uint8
	H2InitialWindow       uint32
	H2WindowUpdateInc     uint32
	H2PseudoOrder         uint16
	H2PseudoOrderCount    uint8
	HTTP1HeaderOrder      [HeaderOrderMax]uint8
	HTTP1HeaderOrderCount uint8
	AcceptLang            []byte
	Body                  []byte
	Origin                []byte
	Host                  []byte
	ContentLength         int
	HasContentLength      bool
	ForceSafe             bool
	Cookie                []byte
	TCPMSS                uint16
	TCPMSSSet             uint8
	TCPTTL                uint8
	TCPTTLSet             uint8
	TCPWindow             uint16
	TCPWindowSet          uint8
	TCPSig                uint32
	TCPSigSet             uint8
	RTTSynMS              uint16
	TTFBAppMS             uint16
	ConnTimingSet         uint8
}

type H2ConnState struct {
	Established            bool
	SettingsSent           bool
	HeaderBlock            []byte
	HeaderStreamID         uint32
	ExpectData             bool
	DataStreamID           uint32
	SettingsScratch        [40]byte
	SettingsLen            int
	IncompleteSpin         uint8
	IncompleteIdleArmed    bool
	IncompleteIdleDeadline int64
	fp                     H2ConnFingerprint
}
