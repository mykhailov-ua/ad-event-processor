package domain

const (
	AntifraudMaxRTTSamples     = 8
	AntifraudMaxChallengeToken = 96
)

// AntifraudSnapshot is the compact server-side view of client antifraud telemetry.
// Populated from /track JSON key "antifraud" by ingest hand-rolled parser.
type AntifraudSnapshot struct {
	NavStartPTMs         uint32
	DwellMs              uint32
	FooterReachMs        uint32
	TrustedRatioMilli    uint16
	PointerCVMilli       uint16
	PointerDtCVMilli     uint16
	ScrollCVMilli        uint16
	ScrollJerkMilli      uint16
	TouchIntervalCVMilli uint16
	RafCvMilli           uint16
	PoWNonce             uint32
	Webdriver            uint8
	AutomationLeak       uint8
	RuntimeLeak          uint8
	ChallengeTokenLen    uint8
	ChallengeToken       [AntifraudMaxChallengeToken]byte
	TelemetryMAC         [32]byte
	CanvasHash           [16]byte
	AudioHash            [16]byte
	WebGLHash            [16]byte
	RTTSamples           [AntifraudMaxRTTSamples]uint16
	RTTSampleCount       uint8
}
