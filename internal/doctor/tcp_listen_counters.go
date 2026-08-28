package doctor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type TCPListenCounters struct {
	ListenOverflows uint64
	ListenDrops     uint64
}

func (c TCPListenCounters) Delta(after TCPListenCounters) TCPListenCounters {
	return TCPListenCounters{
		ListenOverflows: saturatingSub(after.ListenOverflows, c.ListenOverflows),
		ListenDrops:     saturatingSub(after.ListenDrops, c.ListenDrops),
	}
}

func saturatingSub(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}

func ReadTCPListenCounters() (TCPListenCounters, error) {
	f, err := os.Open("/proc/net/netstat")
	if err != nil {
		return TCPListenCounters{}, err
	}
	defer func() { _ = f.Close() }()
	return ParseTCPListenCounters(f)
}

func ParseTCPListenCounters(r io.Reader) (TCPListenCounters, error) {
	values, err := parseTCPExtFieldMap(r)
	if err != nil {
		return TCPListenCounters{}, err
	}
	overflows, ok := values["ListenOverflows"]
	if !ok {
		return TCPListenCounters{}, fmt.Errorf("ListenOverflows not found in /proc/net/netstat")
	}
	drops, ok := values["ListenDrops"]
	if !ok {
		return TCPListenCounters{}, fmt.Errorf("ListenDrops not found in /proc/net/netstat")
	}
	return TCPListenCounters{
		ListenOverflows: overflows,
		ListenDrops:     drops,
	}, nil
}

func parseTCPExtFieldMap(r io.Reader) (map[string]uint64, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var headers []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "TcpExt:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if headers == nil {
			headers = fields[1:]
			continue
		}
		values := fields[1:]
		if len(values) < len(headers) {
			return nil, fmt.Errorf("TcpExt values shorter than header (%d < %d)", len(values), len(headers))
		}
		out := make(map[string]uint64, len(headers))
		for i, name := range headers {
			val, err := strconv.ParseUint(values[i], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse TcpExt %s: %w", name, err)
			}
			out[name] = val
		}
		return out, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("TcpExt counter block not found")
}

func parseTCPListenCountersFromBytes(data []byte) (TCPListenCounters, error) {
	return ParseTCPListenCounters(bytes.NewReader(data))
}
