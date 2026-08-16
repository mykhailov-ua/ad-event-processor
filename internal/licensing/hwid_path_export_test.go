//go:build linux

package licensing

func HWIDPathDMIUUID() uint8          { return hwidPathDMIUUID }
func HWIDPathMachineID() uint8        { return hwidPathMachineID }
func HWIDPathNetPrefix() uint8        { return hwidPathNetPrefix }
func HWIDPathNetAddressSuffix() uint8 { return hwidPathNetAddressSuffix }

func DecodeHWIDPathForTest(pathID uint8, suffix []byte) string {
	var buf [256]byte
	n := decodeHWIDPath(pathID, suffix, buf[:])
	if n <= 0 {
		return ""
	}
	return string(buf[:n])
}

func DecodeHWIDPathFromIDsForTest(pathID uint8, midSuffix []byte, tailID uint8) string {
	return hwidDecodedPathFromIDs(pathID, midSuffix, tailID)
}
