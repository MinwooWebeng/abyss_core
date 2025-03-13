package main

import "C"

const version = "0.9.0"

//export GetVersion
func GetVersion(buf *C.char, buflen C.int) C.int {
	return TryMarshalBytes(buf, buflen, []byte(version))
}

func main() {
	//TODO: debug log initialization
}
