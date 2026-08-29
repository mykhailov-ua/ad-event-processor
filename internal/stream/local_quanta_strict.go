package stream

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type LocalQuantaStrict struct {
	enterMicro int64
	exitMicro  int64
	flags      [localQuantaSlotCount]atomic.Uint32
}

func NewLocalQuantaStrict(enterMicro, exitMicro int64) *LocalQuantaStrict {
	if enterMicro <= 0 {
		enterMicro = 5_000_000
	}
	if exitMicro <= enterMicro {
		exitMicro = enterMicro + 3_000_000
	}
	return &LocalQuantaStrict{
		enterMicro: enterMicro,
		exitMicro:  exitMicro,
	}
}

func (s *LocalQuantaStrict) slotIndex(id uuid.UUID) uint32 {
	return domain.CRC32Castagnoli(&id) & localQuantaSlotMask
}

func (s *LocalQuantaStrict) IsStrict(id uuid.UUID) bool {
	if s == nil {
		return false
	}
	return s.flags[s.slotIndex(id)].Load() == 1
}

func (s *LocalQuantaStrict) UpdateFromRedisRemaining(id uuid.UUID, redisRemaining int64) {
	if s == nil {
		return
	}
	idx := s.slotIndex(id)
	if redisRemaining < s.enterMicro {
		s.flags[idx].Store(1)
		return
	}
	if redisRemaining >= s.exitMicro {
		s.flags[idx].Store(0)
	}
}
