package protocol

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	CmdCommitOffset        uint16 = 5
	CmdCommittedOffset     uint16 = 6
	CmdCommitOffsetResp    uint16 = 105
	CmdCommittedOffsetResp uint16 = 106
	// CommitOffsetRespMetaLen: status u8 + stored_offset u64 (frame body 23, on-wire 27 with length prefix).
	CommitOffsetRespMetaLen = 9
	// CommittedOffsetRespMetaLen: status u8 + committed_offset u64 (same 27-byte response frame).
	CommittedOffsetRespMetaLen = 9

	OffsetStatusOK               byte = 0 // success
	OffsetStatusBadRequest       byte = 1 // malformed key or offset
	OffsetStatusStoreUnavailable byte = 2 // offset store down; client may treat as offset 0
)

// Offset key wire: topic_len u16 + topic + partition u16 + group_len u16 + group (commit adds offset u64).
func DecodeOffsetKeyRequest(payload []byte) (topic string, partition uint16, group string, err error) {
	if len(payload) < 6 {
		return "", 0, "", errors.New("malformed offset request")
	}
	topicLen := int(binary.BigEndian.Uint16(payload[0:2]))
	if len(payload) < 2+topicLen+2+2 {
		return "", 0, "", errors.New("malformed offset request: topic length out of bounds")
	}
	topic = string(payload[2 : 2+topicLen])
	pos := 2 + topicLen
	partition = binary.BigEndian.Uint16(payload[pos : pos+2])
	pos += 2
	groupLen := int(binary.BigEndian.Uint16(payload[pos : pos+2]))
	pos += 2
	if len(payload) < pos+groupLen {
		return "", 0, "", errors.New("malformed offset request: group length out of bounds")
	}
	group = string(payload[pos : pos+groupLen])
	return topic, partition, group, nil
}

func DecodeCommitOffsetRequest(payload []byte) (topic string, partition uint16, group string, offset uint64, err error) {
	topic, partition, group, err = DecodeOffsetKeyRequest(payload)
	if err != nil {
		return "", 0, "", 0, err
	}
	keyLen := 2 + len(topic) + 2 + 2 + len(group)
	if len(payload) < keyLen+8 {
		return "", 0, "", 0, errors.New("malformed commit offset request")
	}
	offset = binary.BigEndian.Uint64(payload[keyLen : keyLen+8])
	return topic, partition, group, offset, nil
}

func EncodeCommitOffsetRequest(buf []byte, seq uint64, topic string, partition uint16, group string, offset uint64) []byte {
	topicBytes := []byte(topic)
	groupBytes := []byte(group)
	framePayloadLen := 2 + len(topicBytes) + 2 + 2 + len(groupBytes) + 8
	totalLen := uint32(2 + 8 + framePayloadLen + 4)

	binary.BigEndian.PutUint32(buf[0:4], totalLen)
	binary.BigEndian.PutUint16(buf[4:6], CmdCommitOffset)
	binary.BigEndian.PutUint64(buf[6:14], seq)
	binary.BigEndian.PutUint16(buf[14:16], uint16(len(topicBytes)))
	copy(buf[16:16+len(topicBytes)], topicBytes)
	pos := 16 + len(topicBytes)
	binary.BigEndian.PutUint16(buf[pos:pos+2], partition)
	pos += 2
	binary.BigEndian.PutUint16(buf[pos:pos+2], uint16(len(groupBytes)))
	pos += 2
	copy(buf[pos:pos+len(groupBytes)], groupBytes)
	pos += len(groupBytes)
	binary.BigEndian.PutUint64(buf[pos:pos+8], offset)

	payloadSlice := buf[14 : 14+framePayloadLen]
	checksum := crc32.ChecksumIEEE(payloadSlice)
	binary.BigEndian.PutUint32(buf[14+framePayloadLen:14+framePayloadLen+4], checksum)
	return buf[:4+2+8+framePayloadLen+4]
}

func EncodeCommittedOffsetRequest(buf []byte, seq uint64, topic string, partition uint16, group string) []byte {
	topicBytes := []byte(topic)
	groupBytes := []byte(group)
	framePayloadLen := 2 + len(topicBytes) + 2 + 2 + len(groupBytes)
	totalLen := uint32(2 + 8 + framePayloadLen + 4)

	binary.BigEndian.PutUint32(buf[0:4], totalLen)
	binary.BigEndian.PutUint16(buf[4:6], CmdCommittedOffset)
	binary.BigEndian.PutUint64(buf[6:14], seq)
	binary.BigEndian.PutUint16(buf[14:16], uint16(len(topicBytes)))
	copy(buf[16:16+len(topicBytes)], topicBytes)
	pos := 16 + len(topicBytes)
	binary.BigEndian.PutUint16(buf[pos:pos+2], partition)
	pos += 2
	binary.BigEndian.PutUint16(buf[pos:pos+2], uint16(len(groupBytes)))
	pos += 2
	copy(buf[pos:pos+len(groupBytes)], groupBytes)

	payloadSlice := buf[14 : 14+framePayloadLen]
	checksum := crc32.ChecksumIEEE(payloadSlice)
	binary.BigEndian.PutUint32(buf[14+framePayloadLen:14+framePayloadLen+4], checksum)
	return buf[:4+2+8+framePayloadLen+4]
}

func EncodeCommitOffsetResponse(buf []byte, seq uint64, status byte, offset uint64) []byte {
	binary.BigEndian.PutUint32(buf[0:4], 23)
	binary.BigEndian.PutUint16(buf[4:6], CmdCommitOffsetResp)
	binary.BigEndian.PutUint64(buf[6:14], seq)
	buf[14] = status
	binary.BigEndian.PutUint64(buf[15:23], offset)
	checksum := crc32.ChecksumIEEE(buf[14:23])
	binary.BigEndian.PutUint32(buf[23:27], checksum)
	return buf[:27]
}

func DecodeCommitOffsetResponse(payload []byte) (status byte, offset uint64, err error) {
	if len(payload) < CommitOffsetRespMetaLen {
		return 0, 0, errors.New("malformed commit offset response")
	}
	status = payload[0]
	offset = binary.BigEndian.Uint64(payload[1:9])
	return status, offset, nil
}

func EncodeCommittedOffsetResponse(buf []byte, seq uint64, status byte, offset uint64) []byte {
	binary.BigEndian.PutUint32(buf[0:4], 23)
	binary.BigEndian.PutUint16(buf[4:6], CmdCommittedOffsetResp)
	binary.BigEndian.PutUint64(buf[6:14], seq)
	buf[14] = status
	binary.BigEndian.PutUint64(buf[15:23], offset)
	checksum := crc32.ChecksumIEEE(buf[14:23])
	binary.BigEndian.PutUint32(buf[23:27], checksum)
	return buf[:27]
}

func DecodeCommittedOffsetResponse(payload []byte) (status byte, offset uint64, err error) {
	if len(payload) < CommittedOffsetRespMetaLen {
		return 0, 0, errors.New("malformed committed offset response")
	}
	status = payload[0]
	offset = binary.BigEndian.Uint64(payload[1:9])
	return status, offset, nil
}

func ValidateConsumerGroup(group string) error {
	if group == "" {
		return errors.New("consumer group is empty")
	}
	if len(group) > 255 {
		return errors.New("consumer group name too long")
	}
	return nil
}
