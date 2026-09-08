package monotime

import _ "unsafe"

//go:linkname Nano runtime.nanotime
func Nano() int64
