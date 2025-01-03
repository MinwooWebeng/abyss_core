package abyss_neighbor_discovery

import (
	"time"

	"github.com/google/uuid"
)

type ANDERROR int

const (
	_      ANDERROR = iota //no error
	EINVAL                 //invalid argument
	EPANIC                 //unrecoverable internal error (must not occur)
)

const DEBUG = true

const (
	JNC_REDUNDANT = 110

	JNC_DUPLICATE = 480
	JNC_CANCELED  = 498
	JNC_CLOSED    = 499

	JNC_COLLISION = 520
	JNC_REJECTED  = 599
)

const (
	JNM_REDUNDANT = "Already Joined"

	JNM_DUPLICATE = "Duplicate Join"
	JNM_CANCELED  = "Join Canceled"
	JNM_CLOSED    = "Peer Disconnected"

	JNM_COLLISION = "Session ID Collided"
	JNM_REJECTED  = "Join Rejected"
)

var INVALID_SESSION_ID = uuid.Max

type PeerSession struct {
	Peer          IANDPeer
	PeerSessionID uuid.UUID
}

// events
type SessionRequest struct {
	LocalSessionID uuid.UUID
	PeerSession
}
type SessionReady struct {
	LocalSessionID uuid.UUID
	PeerSession
}
type SessionClose struct {
	LocalSessionID uuid.UUID
	PeerSession
}
type JoinSuccess struct {
	LocalSessionID uuid.UUID
	WorldURL       string
}
type JoinFail struct {
	LocalSessionID uuid.UUID
	Code           int
	Message        string
}
type ConnectRequest struct {
	AURL string
}
type TimerRequest struct {
	Expiration time.Time
	Payload    any
}
type DebugMessage struct { //only raised in debug mode
	Message string
}

const (
	PRE_MEM_CONNECTED int = iota + 1 //both not accpted
	PRE_MEM_WAITING                  //local accepted
	PRE_MEM_RECVED                   //peer accepted
	PRE_MEM_JN                       //join requested. after jok, PRE_MEM_WAITING
)

type PreMemSession struct {
	state int
	PeerSession
}

// not connected -> [mem recv / accept] -> member
type World struct {
	local_session_id uuid.UUID
	world_url        string
	members          map[string]PeerSession   //id hash - peer session
	pre_members      map[string]PreMemSession //id hash - peer session (MEM sent)
	pre_conn_members map[string]uuid.UUID     //id hash - peer session id (not connected)
	snb_targets      map[string]int           //id hash - left SNB count
}

func NewWorld(local_session_id uuid.UUID, world_url string) *World {
	result := new(World)
	result.local_session_id = local_session_id
	result.world_url = world_url
	result.members = make(map[string]PeerSession)
	result.pre_members = make(map[string]PreMemSession)
	result.pre_conn_members = make(map[string]uuid.UUID)
	result.snb_targets = make(map[string]int)
	return result
}

type WorldCandidate struct {
	members map[string]PeerSession //MEM received (need to respond)
}

func NewWorldCandidate() *WorldCandidate {
	result := new(WorldCandidate)
	result.members = make(map[string]PeerSession)
	return result
}

type JoinTarget struct {
	peer        IANDPeer
	path        string
	pre_members map[string]PeerSession
}

func NewJoinTarget(peer IANDPeer, path string) *JoinTarget {
	result := new(JoinTarget)
	result.peer = peer
	result.path = path
	result.pre_members = make(map[string]PeerSession)
	return result
}

type AND struct {
	EventCh  chan any
	DebugMsg string

	local_hash   string
	peers        map[string]IANDPeer       //id hash - peer
	dead_peers   map[string]IANDPeer       //id hash - peer // Communication failed, but not discarded.
	join_targets map[uuid.UUID]*JoinTarget //local session id - target peer
	worlds       map[uuid.UUID]*World      //local session id - world
}

func NewAND(local_hash string) *AND {
	result := new(AND)
	result.local_hash = local_hash
	result.EventCh = make(chan any, 32)
	result.peers = make(map[string]IANDPeer)
	result.dead_peers = make(map[string]IANDPeer)
	result.join_targets = make(map[uuid.UUID]*JoinTarget)
	result.worlds = make(map[uuid.UUID]*World)
	return result
}

// // drop~ functions must success every time.

// drop the peer from the world (if exists), and raise appropriate event
func (a *AND) dropWorldMember(world *World, peer_hash string) {
	if peer_session, ok := world.members[peer_hash]; ok {
		delete(world.members, peer_hash)
		delete(world.snb_targets, peer_hash)
		if DEBUG {
			if _, ok := world.pre_members[peer_hash]; ok {
				a.EventCh <- DebugMessage{Message: "world pre_MEM_members corrupted - dropWorldMember"}
			}
			if _, ok := world.pre_conn_members[peer_hash]; ok {
				a.EventCh <- DebugMessage{Message: "world pre_conn_members corrupted - dropWorldMember"}
			}
		}
		a.EventCh <- SessionClose{world.local_session_id, peer_session}
	}

	if DEBUG {
		if _, ok := world.snb_targets[peer_hash]; ok {
			a.EventCh <- DebugMessage{Message: "world snb_targets corrupted - dropWorldMember"}
		}
	}

	delete(world.pre_members, peer_hash)
	delete(world.pre_conn_members, peer_hash)
}

func (a *AND) dropPeer(peer IANDPeer) {
	peer_hash := peer.IDHash()
	if DEBUG {
		if _, ok := a.peers[peer_hash]; !ok {
			a.EventCh <- DebugMessage{Message: "peer not found - dropPeer"}
			return
		}
	}
	delete(a.peers, peer_hash)
	a.dead_peers[peer_hash] = peer

	droped_join_targets := make([]uuid.UUID, 0, len(a.join_targets))
	for ls, jt := range a.join_targets {
		if jt.peer.IDHash() == peer_hash {
			droped_join_targets = append(droped_join_targets, ls)
		}
		delete(jt.pre_members, peer_hash)
	}
	for _, ls := range droped_join_targets {
		for _, pre_mem := range a.join_targets[ls].pre_members {
			pre_mem.Peer.TrySendRST(ls, pre_mem.PeerSessionID)
			// we don't handle its failure for now;
			// it is better to be handled, only if we can clearly predict the recursion
			// but practically, no
		}
		delete(a.join_targets, ls)
		a.EventCh <- JoinFail{LocalSessionID: ls, Code: JNC_CLOSED, Message: JNM_CLOSED}
	}
	for _, world := range a.worlds {
		a.dropWorldMember(world, peer_hash)
	}
}

// // clear~ functions removes something, and does related I/O. It never raises event. It must call proper drop action
// drop all world candidates and send RST - kill peers if send fails.
// func (a *AND) clearWorldCandidates() {
// 	dead_peers := make([]IANDPeer, 0, len(a.peers))
// 	for _, world_candidate := range a.world_candidates {
// 		for _, peer_session := range world_candidate.members {
// 			if !peer_session.Peer.TrySendRST(INVALID_SESSION_ID, peer_session.PeerSessionID) {
// 				dead_peers = append(dead_peers, peer_session.Peer)
// 			}
// 		}
// 	}
// 	for u := range a.world_candidates {
// 		delete(a.world_candidates, u)
// 	}

// 	for _, peer := range dead_peers {
// 		a.dropPeer(peer)
// 	}
// }

func (a *AND) ResetOptDrop(peer IANDPeer, local_session_id uuid.UUID, peer_session_id uuid.UUID) bool {
	if !peer.TrySendRST(local_session_id, peer_session_id) {
		a.dropPeer(peer)
		return false
	}
	return true
}

func (a *AND) PeerConnected(peer IANDPeer) ANDERROR {
	peer_hash := peer.IDHash()
	if _, ok := a.peers[peer_hash]; ok {
		a.DebugMsg = "peer already connected"
		return EPANIC
	}
	if _, ok := a.dead_peers[peer_hash]; ok {
		a.DebugMsg = "dead peer reconnected without cleanup"
		return EPANIC
	}
	if a.local_hash == peer_hash {
		a.DebugMsg = "self connection"
		return EPANIC
	}

	a.peers[peer_hash] = peer

	if DEBUG {
		for _, join_target := range a.join_targets {
			if _, ok := join_target.pre_members[peer_hash]; ok {
				a.DebugMsg = "join target corrupted - PeerConnected"
				return EPANIC
			}
		}
	}

	for lssid, world := range a.worlds {
		if DEBUG {
			if _, ok := world.members[peer_hash]; ok {
				a.DebugMsg = "world members corrupted - PeerConnected"
				return EPANIC
			}
			if _, ok := world.snb_targets[peer_hash]; ok {
				a.DebugMsg = "world snb_targets corrupted - PeerConnected"
				return EPANIC
			}
			if _, ok := world.pre_members[peer_hash]; ok {
				a.DebugMsg = "world pre_mem_members corrupted - PeerConnected"
				return EPANIC
			}
		}
		if pre_conn_session_id, ok := world.pre_conn_members[peer_hash]; ok {
			delete(world.pre_conn_members, peer_hash)
			peer_session := PeerSession{Peer: peer, PeerSessionID: pre_conn_session_id}
			world.pre_members[peer_hash] = PreMemSession{state: PRE_MEM_CONNECTED, PeerSession: peer_session}
			a.EventCh <- SessionRequest{LocalSessionID: lssid, PeerSession: peer_session}
		}
	}
	return 0
}

func (a *AND) PeerClose(peer IANDPeer) ANDERROR {
	peer_hash := peer.IDHash()

	if _, ok := a.peers[peer_hash]; ok {
		// alive peer
		a.dropPeer(peer)
		delete(a.dead_peers, peer_hash)
		return 0
	} else if _, ok := a.dead_peers[peer_hash]; ok {
		// dead peer
		delete(a.dead_peers, peer_hash)
		return 0
	} else {
		a.DebugMsg = "closing unknown peer: " + peer_hash
		return EPANIC
	}
}

func (a *AND) OpenWorld(local_session_id uuid.UUID, world_url string) ANDERROR {
	if _, ok := a.join_targets[local_session_id]; ok {
		return EINVAL
	}
	if _, ok := a.worlds[local_session_id]; ok {
		return EINVAL
	}

	a.worlds[local_session_id] = NewWorld(local_session_id, world_url)
	a.EventCh <- JoinSuccess{LocalSessionID: local_session_id, WorldURL: world_url}
	return 0
}

func (a *AND) JoinWorld(local_session_id uuid.UUID, peer IANDPeer, path string) ANDERROR {
	if _, ok := a.peers[peer.IDHash()]; !ok {
		return EINVAL
	}
	if _, ok := a.join_targets[local_session_id]; ok {
		return EINVAL
	}
	if _, ok := a.worlds[local_session_id]; ok {
		return EINVAL
	}

	a.join_targets[local_session_id] = NewJoinTarget(peer, path)
	if !peer.TrySendJN(local_session_id, path) {
		a.dropPeer(peer)
	}
	return 0
}

func (a *AND) CancelJoin(local_session_id uuid.UUID) ANDERROR { //this does not guarantee join cancellation.
	join_target, ok := a.join_targets[local_session_id]
	if !ok {
		return EINVAL
	}
	dead_peers := make([]IANDPeer, 0, len(join_target.pre_members))
	for _, pre_mem := range join_target.pre_members {
		if !pre_mem.Peer.TrySendRST(local_session_id, pre_mem.PeerSessionID) {
			dead_peers = append(dead_peers, pre_mem.Peer)
		}
	}
	for _, dead_peer := range dead_peers {
		a.dropPeer(dead_peer)
	}

	a.EventCh <- JoinFail{LocalSessionID: local_session_id, Code: JNC_CANCELED, Message: JNM_CANCELED}
	delete(a.join_targets, local_session_id)
	return 0
}

func (a *AND) AcceptSession(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR {
	world, ok := a.worlds[local_session_id]
	if !ok {
		return EINVAL
	}

	peer_hash := peer_session.Peer.IDHash()
	pre_member, ok := world.pre_members[peer_hash]
	if !ok || pre_member.PeerSessionID != peer_session.PeerSessionID {
		return EINVAL
	}
	if DEBUG {
		if _, ok := world.members[peer_hash]; ok {
			a.DebugMsg = "world members corrupted - 3"
			return EPANIC
		}
	}
	switch pre_member.state {
	case PRE_MEM_CONNECTED:
		pre_member.state = PRE_MEM_WAITING
		world.pre_members[peer_hash] = pre_member
		if !peer_session.Peer.TrySendMEM(local_session_id, peer_session.PeerSessionID) {
			a.dropPeer(peer_session.Peer)
			return 0
		}
		return 0
	case PRE_MEM_RECVED:
		delete(world.pre_members, peer_hash)
		world.members[peer_hash] = pre_member.PeerSession
		a.EventCh <- SessionReady{local_session_id, peer_session}
		if !peer_session.Peer.TrySendMEM(local_session_id, peer_session.PeerSessionID) {
			a.dropPeer(peer_session.Peer)
			return 0
		}
		return 0
	case PRE_MEM_WAITING:
		return EINVAL
	case PRE_MEM_JN:
		pre_member.state = PRE_MEM_WAITING
		world.pre_members[peer_hash] = pre_member
		member_sessions := make([]PeerSession, 0, len(world.members))
		dead_members := make([]IANDPeer, 0, len(world.members))
		for _, mem := range world.members {
			if !mem.Peer.TrySendJNI(mem.PeerSessionID, peer_session) {
				dead_members = append(dead_members, mem.Peer)
				continue
			}
			member_sessions = append(member_sessions, mem)
		}
		for _, dead_mem := range dead_members {
			a.dropPeer(dead_mem)
		}
		if !peer_session.Peer.TrySendJOK(local_session_id, peer_session.PeerSessionID, world.world_url, member_sessions) {
			a.dropPeer(peer_session.Peer)
			return 0
		}
		return 0
	default:
		return EPANIC
	}
}

func (a *AND) DeclineSession(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR {
	world, ok := a.worlds[local_session_id]
	if !ok {
		return EINVAL
	}

	peer_hash := peer_session.Peer.IDHash()
	pre_mem, ok := world.pre_members[peer_hash]
	if !ok || pre_mem.PeerSessionID != peer_session.PeerSessionID {
		return EINVAL
	}

	switch pre_mem.state {
	case PRE_MEM_CONNECTED:
		delete(world.pre_members, peer_hash)
		return 0
	case PRE_MEM_RECVED:
		delete(world.pre_members, peer_hash)
		a.ResetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return 0
	case PRE_MEM_WAITING:
		return EINVAL //cannot decline already accpted session
	case PRE_MEM_JN:
		delete(world.pre_members, peer_hash)
		if !peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_REJECTED, JNM_REJECTED) {
			a.dropPeer(peer_session.Peer)
		}
		return 0
	default:
		return EPANIC
	}
}

func (a *AND) CloseWorld(local_session_id uuid.UUID) ANDERROR {
	world, ok := a.worlds[local_session_id]
	if !ok {
		return EINVAL
	}

	dead_peers := make([]IANDPeer, 0, len(world.members))
	for _, peer_session := range world.members {
		if !peer_session.Peer.TrySendRST(local_session_id, peer_session.PeerSessionID) {
			dead_peers = append(dead_peers, peer_session.Peer)
		}
		a.EventCh <- SessionClose{
			LocalSessionID: world.local_session_id,
			PeerSession:    peer_session,
		}
	}
	for _, dp := range dead_peers {
		a.dropPeer(dp)
	}

	delete(a.worlds, local_session_id)
	return 0
}

func (a *AND) TimerExpire(payload any) ANDERROR {
	//TODO: SNB
	return 0
}

// session_uuid is always the sender's session id.
func (a *AND) JN(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR {
	world, ok := a.worlds[local_session_id]
	if !ok {
		return EINVAL
	}

	peer_hash := peer_session.Peer.IDHash()
	if _, ok := world.members[peer_hash]; ok {
		if !peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_REDUNDANT, JNM_REDUNDANT) {
			a.dropPeer(peer_session.Peer)
		}
		return 0
	}

	if _, ok := world.pre_members[peer_hash]; ok {
		if !peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_REDUNDANT, JNM_REDUNDANT) {
			a.dropPeer(peer_session.Peer)
		}
		return 0
	}

	// clean join request
	world.pre_members[peer_hash] = PreMemSession{state: PRE_MEM_JN, PeerSession: peer_session}
	a.EventCh <- SessionRequest{LocalSessionID: local_session_id, PeerSession: peer_session}
	return 0
}
func (a *AND) JOK(local_session_id uuid.UUID, peer_session PeerSession, world_url string, member_sessions []PeerSessionInfo) ANDERROR {
	join_target, ok := a.join_targets[local_session_id]
	if !ok {
		a.ResetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return EINVAL
	}
	if join_target.peer != peer_session.Peer {
		a.ResetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return EINVAL
	}

	world := NewWorld(local_session_id, world_url)

	delete(a.join_targets, local_session_id)
	a.worlds[local_session_id] = world
	a.EventCh <- JoinSuccess{local_session_id, world_url}

	world.pre_members[peer_session.Peer.IDHash()] = PreMemSession{state: PRE_MEM_RECVED, PeerSession: peer_session}
	a.EventCh <- SessionRequest{LocalSessionID: local_session_id, PeerSession: peer_session}

	for pre_mem_hash, pre_mem_session := range join_target.pre_members {
		world.pre_members[pre_mem_hash] = PreMemSession{state: PRE_MEM_RECVED, PeerSession: pre_mem_session}
		a.EventCh <- SessionRequest{LocalSessionID: local_session_id, PeerSession: pre_mem_session}
	}

	for _, member_session := range member_sessions {
		if _, ok := world.pre_members[member_session.PeerHash]; ok {
			continue
		}
		world.pre_conn_members[member_session.PeerHash] = member_session.SessionID
		a.EventCh <- ConnectRequest{AURL: member_session.AURL}
	}

	return 0
}
func (a *AND) JDN(local_session_id uuid.UUID, peer IANDPeer, code int, message string) ANDERROR {
	join_target, ok := a.join_targets[local_session_id]
	if !ok {
		return EINVAL
	}
	if join_target.peer != peer {
		return EINVAL
	}

	delete(a.join_targets, local_session_id) // it should never have pre members - what if it does? - IDK :/
	a.EventCh <- JoinFail{local_session_id, code, message}
	return 0
}
func (a *AND) JNI(local_session_id uuid.UUID, peer_session PeerSession, member_hash string, member_aurl string, member_session_id uuid.UUID) ANDERROR {
	world, ok := a.worlds[local_session_id]
	if !ok {
		return EINVAL
	}
	if _, ok := world.members[peer_session.Peer.IDHash()]; !ok {
		a.ResetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return EINVAL
	}

	// this is vulnerable to fabricated JNI flooding
	if _, ok := world.members[member_hash]; ok {
		return 0
	}
	if _, ok := world.pre_members[member_hash]; ok {
		return 0
	}
	if _, ok := world.pre_conn_members[member_hash]; ok {
		return 0
	}

	if member, ok := a.peers[member_hash]; ok {
		member_session := PeerSession{Peer: member, PeerSessionID: member_session_id}
		world.pre_members[member_hash] = PreMemSession{
			state:       PRE_MEM_CONNECTED,
			PeerSession: member_session,
		}
		a.EventCh <- SessionRequest{LocalSessionID: local_session_id, PeerSession: member_session}
		return 0
	}

	world.pre_conn_members[member_hash] = member_session_id
	a.EventCh <- ConnectRequest{AURL: member_aurl}
	return 0
}
func (a *AND) MEM(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR {
	peer_hash := peer_session.Peer.IDHash()

	world, ok := a.worlds[local_session_id]
	if !ok {
		// not world-related. search for join targets
		join_target, ok := a.join_targets[local_session_id]
		if !ok {
			// not join target either. ignore
			return EINVAL
		}

		if old_session, ok := join_target.pre_members[peer_hash]; ok {
			if old_session.PeerSessionID != peer_session.PeerSessionID {
				if !a.ResetOptDrop(old_session.Peer, local_session_id, old_session.PeerSessionID) {
					return 0
				}
			}
			a.ResetOptDrop(old_session.Peer, local_session_id, peer_session.PeerSessionID)
			return 0
		}

		// add pre_member in join target
		join_target.pre_members[peer_hash] = peer_session
		return 0
	}

	if old_session, ok := world.members[peer_hash]; ok {
		a.dropWorldMember(world, peer_hash)

		if old_session.PeerSessionID != peer_session.PeerSessionID {
			if !a.ResetOptDrop(old_session.Peer, local_session_id, old_session.PeerSessionID) {
				return 0
			}
		}
		a.ResetOptDrop(old_session.Peer, local_session_id, peer_session.PeerSessionID)
		return 0
	}

	if pre_mem, ok := world.pre_members[peer_hash]; ok {
		switch pre_mem.state {
		case PRE_MEM_CONNECTED:
			if pre_mem.PeerSessionID != peer_session.PeerSessionID {
				// peer session mismatch with expection
				delete(world.pre_members, peer_hash)
				if !a.ResetOptDrop(pre_mem.Peer, local_session_id, pre_mem.PeerSessionID) {
					return 0
				}
				a.ResetOptDrop(pre_mem.Peer, local_session_id, peer_session.PeerSessionID)
				return 0
			}
			// expected MEM
			pre_mem.state = PRE_MEM_RECVED
			world.pre_members[peer_hash] = pre_mem

		case PRE_MEM_WAITING:
			if pre_mem.PeerSessionID != peer_session.PeerSessionID {
				// peer session mismatch with expection
				delete(world.pre_members, peer_hash)
				if !a.ResetOptDrop(pre_mem.Peer, local_session_id, pre_mem.PeerSessionID) {
					return 0
				}
				a.ResetOptDrop(pre_mem.Peer, local_session_id, peer_session.PeerSessionID)
				return 0
			}
			// expected MEM
			delete(world.pre_members, peer_hash)
			world.members[peer_hash] = pre_mem.PeerSession
			a.EventCh <- SessionReady{LocalSessionID: local_session_id, PeerSession: pre_mem.PeerSession}

		case PRE_MEM_RECVED, PRE_MEM_JN:
			// 1) duplicate MEM - reset both

			// 2) MEM after JN
			// A -- C, B -JN> C, A -JN> B, C -JNI(B)> A, A -MEM> B. B has not accepted yet.
			// resetting both is most clean

			delete(world.pre_members, peer_hash)
			if pre_mem.PeerSessionID != peer_session.PeerSessionID {
				if !a.ResetOptDrop(pre_mem.Peer, local_session_id, pre_mem.PeerSessionID) {
					return 0
				}
			}
			a.ResetOptDrop(pre_mem.Peer, local_session_id, peer_session.PeerSessionID)
		}
		return 0
	}

	world.pre_members[peer_hash] = PreMemSession{state: PRE_MEM_RECVED, PeerSession: peer_session}
	return 0
}

// TODO
func (a *AND) SNB(local_session_id uuid.UUID, peer_session PeerSession, member_hashes []string) ANDERROR {
	return 0
}
func (a *AND) CRR(local_session_id uuid.UUID, peer_session PeerSession, member_hashes []string) ANDERROR {
	return 0
}
func (a *AND) RST(local_session_id uuid.UUID, peer_session PeerSession) ANDERROR {
	return 0
}

// debug
func (a *AND) PrintReport() string {
	return "[AND debug report]"
}
