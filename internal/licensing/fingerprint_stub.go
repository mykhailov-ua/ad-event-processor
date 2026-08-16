//go:build !linux

package licensing

func readMachineID() string {
	return ""
}
