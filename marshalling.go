package main

import "C"
import (
	"os"
	"time"
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

func DebugLog(content string) {
	file, err := os.OpenFile("abyssnet_dll_crash_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	file.WriteString(time.Now().Format("00:00:00.000000") + " " + content)
	file.Close()
}
