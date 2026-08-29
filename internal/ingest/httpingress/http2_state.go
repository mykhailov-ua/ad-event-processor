package httpingress

func NewH2ConnState() H2ConnState {
	return H2ConnState{
		HeaderBlock: make([]byte, 0, 256),
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
	if len(s.HeaderBlock)+len(p) > h2MaxHeaderBlock {
		return ErrInvalid
	}
	s.HeaderBlock = append(s.HeaderBlock, p...)
	return nil
}
