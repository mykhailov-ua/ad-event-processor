package domain

const (
	UDPMagic            = udpMagic
	UDPProtocolVersion  = udpProtocolVersion
	UDPProtocolVersion2 = udpProtocolVersion2
)

func UDPShardPayloadLen(numShards uint8) int {
	return udpShardPayloadLen(numShards)
}

func UDPDecodeHeader(src []byte, hdr *UDPHeader) bool {
	return udpDecodeHeader(src, hdr)
}

func UDPDecodeShardLimits(payload []byte, numShards uint8, version uint8, out *UDPControlLimits) bool {
	return udpDecodeShardLimits(payload, numShards, version, out)
}

func UDPDecodeNodeWeights(payload []byte, out *[]UDPNodeWeight) bool {
	return udpDecodeNodeWeights(payload, out)
}

func UDPEncodeHeader(dst []byte, hdr *UDPHeader) int {
	return udpEncodeHeader(dst, hdr)
}

func UDPEncodeConfigRequest(dst []byte, req *UDPConfigRequestPayload) int {
	return udpEncodeConfigRequest(dst, req)
}

func UDPEncodeShardLimits(dst []byte, limits *UDPControlLimits) int {
	return udpEncodeShardLimits(dst, limits)
}

func UDPEncodeNodeWeights(dst []byte, weights []UDPNodeWeight) int {
	return udpEncodeNodeWeights(dst, weights)
}

func UDPApplyCanaryFloor(limits *UDPControlLimits) {
	udpApplyCanaryFloor(limits)
}

func UDPLimitsTightening(prev, next *UDPControlLimits) bool {
	return udpLimitsTightening(prev, next)
}
