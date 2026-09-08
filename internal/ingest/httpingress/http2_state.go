package httpingress

func NewH2ConnState() H2ConnState {
	return H2ConnState{
		HeaderBlock: make([]byte, 0, 1024),
	}
}

func (s *H2ConnState) ResetConn() {
	s.Established = false
	s.SettingsSent = false
	s.SettingsLen = 0
	s.IncompleteSpin = 0
	s.IncompleteIdleArmed = false
	s.IncompleteIdleDeadline = 0
	s.fp = H2ConnFingerprint{}
	s.ResetStream()
}

func (s *H2ConnState) ResetStream() {
	s.HeaderBlock = s.HeaderBlock[:0]
	s.ExpectData = false
	s.DataStreamID = 0
	s.HeaderStreamID = 0
}

func (s *H2ConnState) appendSettingsOut(extra []byte) []byte {
	s.SettingsLen += copy(s.SettingsScratch[s.SettingsLen:], extra)
	return s.SettingsScratch[:s.SettingsLen]
}

const h2MaxHeaderBlock = 16 << 10

func (s *H2ConnState) appendHeaderBlock(p []byte) error {
	need := len(s.HeaderBlock) + len(p)
	if need > h2MaxHeaderBlock {
		return ErrInvalid
	}
	if cap(s.HeaderBlock) < need {
		grow := need
		if grow < 256 {
			grow = 256
		}
		if grow > h2MaxHeaderBlock {
			grow = h2MaxHeaderBlock
		}
		next := make([]byte, len(s.HeaderBlock), grow)
		copy(next, s.HeaderBlock)
		s.HeaderBlock = next
	}
	s.HeaderBlock = append(s.HeaderBlock, p...)
	return nil
}
