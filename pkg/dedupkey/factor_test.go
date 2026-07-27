package dedupkey

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalProxyBatchPayload_StableSort(t *testing.T) {
	payloads := [][]byte{
		[]byte(`{"z":1}`),
		[]byte(`{"a":1}`),
		[]byte(`{"m":1}`),
	}
	seqs := []uint64{2, 0, 1}

	type pair struct {
		seq     uint64
		payload []byte
	}
	records := make([]pair, len(seqs))
	for i := range seqs {
		records[i] = pair{seq: seqs[i], payload: payloads[i]}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].seq < records[j].seq
	})

	var buf [512]byte
	var merged []byte
	for _, rec := range records {
		merged = WriteCanonicalProxyBatchPayload(merged[:0], rec.seq, rec.payload)
		merged = append(merged, ';')
	}

	shuffled := []pair{
		{seq: 1, payload: payloads[2]},
		{seq: 2, payload: payloads[0]},
		{seq: 0, payload: payloads[1]},
	}
	sort.Slice(shuffled, func(i, j int) bool {
		return shuffled[i].seq < shuffled[j].seq
	})
	var merged2 []byte
	for _, rec := range shuffled {
		merged2 = WriteCanonicalProxyBatchPayload(merged2[:0], rec.seq, rec.payload)
		merged2 = append(merged2, ';')
	}
	assert.Equal(t, merged, merged2)

	u1 := FactorU(WriteCanonicalProxyBatchPayload(buf[:0], 9, payloads[1]))
	u2 := FactorU(WriteCanonicalProxyBatchPayload(buf[:0], 9, payloads[1]))
	assert.Equal(t, u1, u2)
}
