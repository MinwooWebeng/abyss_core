package main

import "C"
import (
	"runtime/cgo"
	"unsafe"

	abyss "abyss_neighbor_discovery/interfaces"
)

func TryMarshalBytes(buf *C.char, buflen C.int, data []byte) C.int {
	if int(buflen) < len(data) {
		return -1
	}

	slice := (*[1 << 28]byte)(unsafe.Pointer(buf))[:buflen]
	copy(slice, data)
	return C.int(len(data))
}

func UnmarshalBytes(buf *C.char, buflen C.int) []byte {
	return (*[1 << 28]byte)(unsafe.Pointer(buf))[:buflen]
}

func MarshalAbyssHost(a abyss.IAbyssHost) cgo.Handle {
	return cgo.NewHandle(a)
}

func UnmarshalAbyssHost(h cgo.Handle) abyss.IAbyssHost {
	return h.Value().(abyss.IAbyssHost)
}
