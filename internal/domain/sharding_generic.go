//go:build !amd64

package domain

import (
	"hash/crc32"

	"github.com/google/uuid"
)

var crc32CastagnoliTable = crc32.MakeTable(crc32.Castagnoli)

func crc32Castagnoli(data *uuid.UUID) uint32 {
	return crc32.Checksum(data[:], crc32CastagnoliTable)
}
