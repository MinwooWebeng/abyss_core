package main

import "C"
import (
	"unsafe"
)

func TryMarshalBytes(buf *C.char, buflen C.int, data []byte) C.int {
	if int(buflen) < len(data) {
		return BUFFER_OVERFLOW
	}

	slice := (*[1 << 28]byte)(unsafe.Pointer(buf))[:buflen]
	copy(slice, data)
	return C.int(len(data))
}

func UnmarshalBytes(buf *C.char, buflen C.int) []byte {
	return (*[1 << 28]byte)(unsafe.Pointer(buf))[:buflen]
}
