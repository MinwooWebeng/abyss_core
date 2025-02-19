package message

// import "io"

type FrameType int

// abyss neighbor discovery
const (
	_ FrameType = iota
	JN
	JOK
	JDN
	JNI
	MEM
	SNB
	CRR
	RST
)

// shared object model
const (
	SOR FrameType = iota + 20 //request
	SO                        //init
	SOA                       //new
	SOD                       //delete
)

// ping
const (
	PINGT FrameType = iota + 60
	PINGR
)

type IDFrame struct { // delivers real identity -
	Payload []byte //certificate - local id -> what? metadata?
}

type DummyAuth struct {
	Name string
}
