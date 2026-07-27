package opkey

import "testing"

func BenchmarkPoolEnqueueDequeue(b *testing.B) {
	q := NewMPSCQueue(4096)
	var slot Slot
	slot.Seq = 1
	slot.setDerived()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for !q.Push(&slot) {
		}
		for {
			if _, ok := q.Pop(); ok {
				break
			}
		}
	}
}
