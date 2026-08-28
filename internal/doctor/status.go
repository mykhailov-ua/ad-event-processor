package doctor

type Status int

const (
	StatusPass Status = iota
	StatusWarn
	StatusFail
	StatusSkip
)

func (s Status) String() string {
	switch s {
	case StatusPass:
		return "pass"
	case StatusWarn:
		return "warn"
	case StatusFail:
		return "fail"
	case StatusSkip:
		return "skip"
	default:
		return "unknown"
	}
}

type Result struct {
	Name    string
	Status  Status
	Detail  string
	Latency int64
}

type Report struct {
	Results []Result
}

func (r Report) ExitCode() int {
	hasFail := false
	hasWarn := false
	for _, res := range r.Results {
		switch res.Status {
		case StatusFail:
			hasFail = true
		case StatusWarn:
			hasWarn = true
		}
	}
	if hasFail {
		return 2
	}
	if hasWarn {
		return 1
	}
	return 0
}
