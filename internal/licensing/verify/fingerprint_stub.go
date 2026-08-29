//go:build !linux

package verify

func readMachineID() string {
	return ""
}
