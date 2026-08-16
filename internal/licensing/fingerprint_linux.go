//go:build linux

package licensing

func readMachineID() string {
	if id := readHWIDFile(hwidPathMachineID, nil); id != "" {
		return id
	}
	return readHWIDFile(hwidPathDBusMachineID, nil)
}
