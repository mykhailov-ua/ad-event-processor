package ingestion

import "ad-event-processor/internal/domain"

func udpShardPayloadLen(numShards uint8) int {
	return domain.UDPShardPayloadLen(numShards)
}

func udpDecodeHeader(src []byte, hdr *UDPHeader) bool {
	return domain.UDPDecodeHeader(src, hdr)
}

func udpDecodeShardLimits(payload []byte, numShards uint8, version uint8, out *UDPControlLimits) bool {
	return domain.UDPDecodeShardLimits(payload, numShards, version, out)
}

func udpDecodeNodeWeights(payload []byte, out *[]UDPNodeWeight) bool {
	return domain.UDPDecodeNodeWeights(payload, out)
}

func udpEncodeHeader(dst []byte, hdr *UDPHeader) int {
	return domain.UDPEncodeHeader(dst, hdr)
}

func udpEncodeConfigRequest(dst []byte, req *UDPConfigRequestPayload) int {
	return domain.UDPEncodeConfigRequest(dst, req)
}

func udpEncodeShardLimits(dst []byte, limits *UDPControlLimits) int {
	return domain.UDPEncodeShardLimits(dst, limits)
}

func udpEncodeNodeWeights(dst []byte, weights []UDPNodeWeight) int {
	return domain.UDPEncodeNodeWeights(dst, weights)
}

func udpApplyCanaryFloor(limits *UDPControlLimits) {
	domain.UDPApplyCanaryFloor(limits)
}

func udpLimitsTightening(prev, next *UDPControlLimits) bool {
	return domain.UDPLimitsTightening(prev, next)
}

const (
	udpMagic            = domain.UDPMagic
	udpProtocolVersion  = domain.UDPProtocolVersion
	udpProtocolVersion2 = domain.UDPProtocolVersion2
	udpProtocolVersion3 = domain.UDPProtocolVersion3
)
