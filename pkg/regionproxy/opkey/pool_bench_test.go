package opkey

import "testing"

func BenchmarkPoolEnqueueDequeue(b *testing.B) {
	q := NewMPSCQueue(4096)
	var slot Slot
	slot.Seq = 1
	slot.setDerived()

	b.ReportAllocs()
	for b.Loop() {
		for !q.Push(&slot) {
		}
		for {
			if _, ok := q.Pop(); ok {
				break
			}
		}
	}
}
