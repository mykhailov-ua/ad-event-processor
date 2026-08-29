//go:build linux

package verify

func readMachineID() string {
	if id := readHWIDFile(hwidPathMachineID, nil); id != "" {
		return id
	}
	return readHWIDFile(hwidPathDBusMachineID, nil)
}
