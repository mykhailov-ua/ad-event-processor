package antifraudtelemetry

const (
	AutomationLeakCDC           = 1 << 0
	AutomationLeakPhantom       = 1 << 1
	AutomationLeakPlaywright    = 1 << 2
	AutomationLeakSelenium      = 1 << 3
	AutomationLeakWebdriverAPI  = 1 << 4
	AutomationLeakNativeTamper  = 1 << 5
	AutomationLeakHeadlessShell = 1 << 6
	AutomationLeakStackProbe    = 1 << 7
)

// Input is the server-side scoring view after ingest normalizes client JSON.
type Input struct {
	NavStartPTMs         uint32
	DwellMs              uint32
	FooterReachMs        uint32
	TrustedRatioMilli    uint16
	PointerCVMilli       uint16
	PointerDtCVMilli     uint16
	ScrollCVMilli        uint16
	ScrollJerkMilli      uint16
	TouchIntervalCVMilli uint16
	Webdriver            uint8
	AutomationLeak       uint8
	RuntimeLeak          uint8
	RafCvMilli           uint16
	RTTSamples           []uint16
	ServerRTTSynMS       uint16
	ServerTTFBMS         uint16
	ConnTimingSet        uint8
}

// Verdict holds independent anomaly flags; filter maps each to a fraud reason.
type Verdict struct {
	AutomationLeak   bool
	TemplateBehavior bool
	FastProbe        bool
	UntrustedEvents  bool
	ProxyJitter      bool
	EmptyKinematics  bool
	RttMissing       bool
}

func Score(in Input) Verdict {
	var v Verdict
	if in.Webdriver != 0 || in.AutomationLeak != 0 || in.RuntimeLeak != 0 {
		v.AutomationLeak = true
	}
	if in.RafCvMilli > 800 {
		v.AutomationLeak = true
	}
	// isTrusted alone is bypassable via CDP Input.dispatch*; pair low ratio with metronomic dt CV.
	if in.TrustedRatioMilli > 0 && in.TrustedRatioMilli < 400 {
		if in.PointerDtCVMilli > 0 && in.PointerDtCVMilli < 35 {
			v.UntrustedEvents = true
		} else if in.PointerCVMilli > 0 && in.PointerCVMilli < 45 {
			v.UntrustedEvents = true
		}
	}
	v.TemplateBehavior = scoreTemplateBehavior(in)
	v.FastProbe = scoreFastProbe(in)
	v.ProxyJitter = scoreProxyJitter(in)
	v.EmptyKinematics = scoreEmptyKinematics(in)
	v.RttMissing = scoreRttMissing(in)
	return v
}

func scoreEmptyKinematics(in Input) bool {
	if in.DwellMs <= 500 {
		return false
	}
	return in.PointerCVMilli == 0 &&
		in.PointerDtCVMilli == 0 &&
		in.ScrollCVMilli == 0 &&
		in.TouchIntervalCVMilli == 0
}

func scoreRttMissing(in Input) bool {
	return in.DwellMs > 2000 && len(in.RTTSamples) == 0
}

// scoreTemplateBehavior flags monotonic checker sessions: near-zero kinematic variance with scripted scroll.
func scoreTemplateBehavior(in Input) bool {
	if in.DwellMs == 0 {
		return false
	}
	lowPointerVar := in.PointerCVMilli > 0 && in.PointerCVMilli < 45
	lowScrollVar := in.ScrollCVMilli > 0 && in.ScrollCVMilli < 35
	lowTouchVar := in.TouchIntervalCVMilli > 0 && in.TouchIntervalCVMilli < 40
	lowJerk := in.ScrollJerkMilli > 0 && in.ScrollJerkMilli < 25
	metronomicPointer := in.PointerDtCVMilli > 0 && in.PointerDtCVMilli < 25 && lowPointerVar
	if metronomicPointer {
		return true
	}
	if lowPointerVar && lowScrollVar && lowJerk {
		return true
	}
	if lowTouchVar && lowScrollVar && in.DwellMs < 4000 {
		return true
	}
	// Footer reached quickly with uniform pointer path suggests template crawl.
	if in.FooterReachMs > 0 && in.FooterReachMs < 2500 && lowPointerVar && lowScrollVar {
		return true
	}
	return false
}

// scoreFastProbe detects sub-second attention with deep scroll (catalog scraper pacing).
func scoreFastProbe(in Input) bool {
	if in.DwellMs == 0 || in.FooterReachMs == 0 {
		return false
	}
	if in.DwellMs < 900 && in.FooterReachMs <= in.DwellMs {
		return true
	}
	if in.NavStartPTMs > 0 && in.FooterReachMs > 0 && in.FooterReachMs < 1200 && in.PointerCVMilli < 60 {
		return true
	}
	return false
}

// scoreProxyJitter compares client RTT samples with edge conn timing to catch oscillating proxy hops.
func scoreProxyJitter(in Input) bool {
	samples := in.RTTSamples
	if len(samples) < 3 {
		return false
	}
	var sum uint64
	for _, s := range samples {
		sum += uint64(s)
	}
	mean := float64(sum) / float64(len(samples))
	if mean <= 0 {
		return false
	}
	var varSum float64
	for _, s := range samples {
		d := float64(s) - mean
		varSum += d * d
	}
	std := 0.0
	if len(samples) > 1 {
		std = varSum / float64(len(samples)-1)
		if std > 0 {
			std = sqrt(std)
		}
	}
	cvMilli := uint16((std / mean) * 1000)
	if cvMilli > 420 && mean < 140 {
		return true
	}
	p50 := percentileUint16(samples, 50)
	p90 := percentileUint16(samples, 90)
	if p50 > 0 && p90 > p50*3 && p50 < 110 {
		return true
	}
	if in.ConnTimingSet != 0 && in.ServerRTTSynMS > 0 && p50 > 0 {
		delta := int(p50) - int(in.ServerRTTSynMS)
		if delta < 0 {
			delta = -delta
		}
		if delta > 45 && cvMilli > 250 {
			return true
		}
	}
	if in.ServerTTFBMS > 0 && p50 > 0 && in.ServerTTFBMS > p50*4 && cvMilli > 300 {
		return true
	}
	return false
}

func percentileUint16(samples []uint16, pct int) uint16 {
	if len(samples) == 0 {
		return 0
	}
	buf := make([]uint16, len(samples))
	copy(buf, samples)
	for i := 1; i < len(buf); i++ {
		v := buf[i]
		j := i - 1
		for j >= 0 && buf[j] > v {
			buf[j+1] = buf[j]
			j--
		}
		buf[j+1] = v
	}
	idx := (len(buf) - 1) * pct / 100
	return buf[idx]
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for range 8 {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
