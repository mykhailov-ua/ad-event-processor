package ingestion

import (
	"errors"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"
)

const ProtoMaxFields = 256

var (
	protoMaxFields      = ProtoMaxFields
	errProtoFieldBudget = errors.New("protobuf field budget exceeded")
)

func configureProtoMaxFields(cfg *config.Config) {
	if cfg == nil || cfg.ProtoMaxFields <= 0 {
		protoMaxFields = ProtoMaxFields
		return
	}
	protoMaxFields = cfg.ProtoMaxFields
}

func unmarshalAdEventVT(evt *pb.AdEvent, wire []byte) error {
	if evt == nil {
		return errProtoFieldBudget
	}
	if _, err := protoWireFieldCount(wire, protoMaxFields); err != nil {
		return err
	}
	return evt.UnmarshalVT(wire)
}

func protoWireFieldCount(wire []byte, maxFields int) (int, error) {
	off := 0
	n := len(wire)
	count := 0
	for off < n {
		tag, next, err := protoDecodeVarint(wire, off)
		if err != nil {
			return count, err
		}
		if tag == 0 {
			return count, errProtoFieldBudget
		}
		off = next
		count++
		if count > maxFields {
			return count, errProtoFieldBudget
		}
		wireType := tag & 7
		fieldNum := tag >> 3
		if fieldNum == 0 {
			return count, errProtoFieldBudget
		}
		switch wireType {
		case 0:
			_, off, err = protoDecodeVarint(wire, off)
		case 1:
			off += 8
			if off > n {
				return count, errProtoFieldBudget
			}
		case 2:
			var ln uint64
			ln, off, err = protoDecodeVarint(wire, off)
			if err != nil {
				return count, err
			}
			if ln > uint64(n-off) {
				return count, errProtoFieldBudget
			}
			off += int(ln)
		case 5:
			off += 4
			if off > n {
				return count, errProtoFieldBudget
			}
		default:
			return count, errProtoFieldBudget
		}
		if err != nil || off > n {
			return count, errProtoFieldBudget
		}
	}
	return count, nil
}

func protoDecodeVarint(wire []byte, off int) (uint64, int, error) {
	n := len(wire)
	if off >= n {
		return 0, off, errProtoFieldBudget
	}
	var val uint64
	shift := uint(0)
	for i := off; i < n; i++ {
		b := wire[i]
		if shift >= 64 {
			return 0, off, errProtoFieldBudget
		}
		val |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return val, i + 1, nil
		}
		shift += 7
	}
	return 0, off, errProtoFieldBudget
}

// chaosProtoWireFieldFlood builds a wire with n varint fields (unknown tags).
func chaosProtoWireFieldFlood(n int) []byte {
	wire := make([]byte, 0, n*4)
	for i := 0; i < n; i++ {
		tag := uint64((i%200 + 1) << 3)
		wire = appendProtoVarint(wire, tag)
		wire = appendProtoVarint(wire, 1)
	}
	return wire
}

func appendProtoVarint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}
