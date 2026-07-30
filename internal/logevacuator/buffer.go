package logevacuator

const copyBufferSize = 32 * 1024

func copyBuffer() []byte {
	return make([]byte, copyBufferSize)
}
