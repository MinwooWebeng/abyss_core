package and

import (
	"github.com/google/uuid"

	abyss "github.com/MinwooWebeng/abyss_core/interfaces"
)

const (
	PRE_MEM_CONNECTED int = iota + 1 //both not accpted
	PRE_MEM_WAITING                  //local accepted
	PRE_MEM_RECVED                   //peer accepted
	PRE_MEM_JN                       //join requested. after jok, PRE_MEM_WAITING
)

type PreMemSession struct {
	state int
	abyss.ANDPeerSession
}

type World struct {
	local_session_id uuid.UUID
	world_url        string
	members          map[string]abyss.ANDPeerSession //id hash - peer session
	pre_members      map[string]PreMemSession        //id hash - peer session (MEM sent)
	pre_conn_members map[string]uuid.UUID            //id hash - peer session id (not connected)
	snb_targets      map[string]int                  //id hash - left SNB count
}

func NewWorld(local_session_id uuid.UUID, world_url string) *World {
	result := new(World)
	result.local_session_id = local_session_id
	result.world_url = world_url
	result.members = make(map[string]abyss.ANDPeerSession)
	result.pre_members = make(map[string]PreMemSession)
	result.pre_conn_members = make(map[string]uuid.UUID)
	result.snb_targets = make(map[string]int)
	return result
}

type WorldCandidate struct {
	members map[string]abyss.ANDPeerSession //MEM received (need to respond)
}

func NewWorldCandidate() *WorldCandidate {
	result := new(WorldCandidate)
	result.members = make(map[string]abyss.ANDPeerSession)
	return result
}

type JoinTarget struct {
	peer        abyss.IANDPeer
	path        string
	pre_members map[string]abyss.ANDPeerSession
}

func NewJoinTarget(peer abyss.IANDPeer, path string) *JoinTarget {
	result := new(JoinTarget)
	result.peer = peer
	result.path = path
	result.pre_members = make(map[string]abyss.ANDPeerSession)
	return result
}

const (
	JNC_REDUNDANT = 110

	//Joiner-side problem
	JNC_DUPLICATE = 480
	JNC_CANCELED  = 498
	JNC_CLOSED    = 499

	//Accepter-side response
	JNC_COLLISION = 520
	JNC_RESET     = 598
	JNC_REJECTED  = 599
)

const (
	JNM_REDUNDANT = "Already Joined"

	JNM_DUPLICATE = "Duplicate Join"
	JNM_CANCELED  = "Join Canceled"
	JNM_CLOSED    = "Peer Disconnected"

	JNM_COLLISION = "Session ID Collided"
	JNM_RESET     = "Reset Requested"
	JNM_REJECTED  = "Join Rejected"
)
