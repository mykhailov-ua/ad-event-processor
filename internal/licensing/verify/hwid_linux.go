//go:build linux

package verify

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

func collectHWIDTelemetry() HWIDTelemetry {
	tel := HWIDTelemetry{
		DMIUUID:  readDMIUUID(),
		DiskID:   readRootDiskID(),
		MAC:      readFirstNICMAC(),
		CPUModel: readCPUModel(),
		CPUCores: runtime.NumCPU(),
	}
	if HWIDV3Enabled() {
		tel.MachineID = readMachineID()
	}
	return tel
}

func readDMIUUID() string {
	return readHWIDFile(hwidPathDMIUUID, nil)
}

func readRootDiskID() string {
	if id := readMountSourceUUID(hwidPathMountInfo); id != "" {
		return id
	}
	if id := readMountSourceUUID(hwidPathMounts); id != "" {
		return id
	}
	dev := readRootBlockDevice()
	if dev == "" {
		return ""
	}
	if serial := readBlockSerial(dev); serial != "" {
		return serial
	}
	return dev
}

func readMountSourceUUID(pathID uint8) string {
	path := hwidDecodedPath(pathID, nil)
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, " / ") && !strings.HasSuffix(line, " /") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		src := fields[1]
		if strings.HasPrefix(src, "UUID=") {
			return strings.TrimPrefix(src, "UUID=")
		}
		if strings.HasPrefix(src, "/dev/") {
			if serial := readBlockSerial(src); serial != "" {
				return serial
			}
			return src
		}
	}
	if sc.Err() != nil {
		return ""
	}
	return ""
}

func readRootBlockDevice() string {
	path := hwidDecodedPath(hwidPathMountInfo, nil)
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 5 {
			continue
		}
		mountPoint := fields[4]
		if mountPoint != "/" {
			continue
		}
		if len(fields) < 6 {
			return ""
		}
		return fields[5]
	}
	if sc.Err() != nil {
		return ""
	}
	return ""
}

func readBlockSerial(devPath string) string {
	base := pathBaseName(devPath)
	if base == "" || base == "." {
		return ""
	}
	block := base
	if strings.HasPrefix(base, "nvme") && strings.Contains(base, "p") {
		block = strings.SplitN(base, "p", 2)[0]
	} else if len(base) > 3 && base[len(base)-1] >= '0' && base[len(base)-1] <= '9' {
		block = strings.TrimRight(base, "0123456789")
	}
	if serial := hwidReadStringFromIDs(hwidPathBlockPrefix, []byte(block), hwidPathBlockDeviceSerial); serial != "" {
		return serial
	}
	return hwidReadStringFromIDs(hwidPathBlockPrefix, []byte(block), hwidPathBlockSerial)
}

func readFirstNICMAC() string {
	dir := hwidDecodedPath(hwidPathNetDir, nil)
	if dir == "" {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, ent := range entries {
		name := ent.Name()
		if name == "lo" || strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "br-") {
			continue
		}
		mac := hwidReadStringFromIDs(hwidPathNetPrefix, []byte(name), hwidPathNetAddressSuffix)
		mac = strings.TrimSpace(mac)
		if mac != "" && mac != "00:00:00:00:00:00" {
			return mac
		}
	}
	return ""
}

func readCPUModel() string {
	path := hwidDecodedPath(hwidPathCPUInfo, nil)
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "model name") {
			continue
		}
		_, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		return strings.TrimSpace(val)
	}
	if sc.Err() != nil {
		return ""
	}
	return ""
}

func pathBaseName(path string) string {
	if path == "" {
		return ""
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return path
	}
	return path[i+1:]
}
