package gnet

var (
	respWorkerPoolOverload = []byte("HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain\r\nRetry-After: 1\r\nContent-Length: 17\r\nConnection: keep-alive\r\n\r\nserver overloaded")
	respBadRequestClose    = []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	respPayloadTooLarge    = []byte("HTTP/1.1 413 Payload Too Large\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
)
