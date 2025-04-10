package and

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	"abyss_core/aurl"
	abyss "abyss_core/interfaces"
)

const DEBUG = true

type AND struct {
	eventCh chan abyss.NeighborEvent

	local_hash              string
	peers                   map[string]abyss.IANDPeer //id hash - peer
	dead_peers              map[string]abyss.IANDPeer //id hash - peer // Communication failed, but not discarded.
	join_targets_connecting map[uuid.UUID]*aurl.AURL  //local session id - target peer id hash
	join_targets            map[uuid.UUID]*JoinTarget //local session id - target peer
	worlds                  map[uuid.UUID]*World      //local session id - world

	api_mtx *sync.Mutex
}

func NewAND(local_hash string) *AND {
	return &AND{
		eventCh:                 make(chan abyss.NeighborEvent, 64),
		local_hash:              local_hash,
		peers:                   make(map[string]abyss.IANDPeer),
		dead_peers:              make(map[string]abyss.IANDPeer),
		join_targets_connecting: make(map[uuid.UUID]*aurl.AURL),
		join_targets:            make(map[uuid.UUID]*JoinTarget),
		worlds:                  make(map[uuid.UUID]*World),
		api_mtx:                 new(sync.Mutex),
	}
}

func (a *AND) EventChannel() chan abyss.NeighborEvent {
	return a.eventCh
}

func (a *AND) PeerConnected(peer abyss.IANDPeer) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	peer_hash := peer.IDHash()
	//fmt.Println("peer connected: " + peer_hash)
	if _, ok := a.peers[peer_hash]; ok {
		a.eventCh <- abyss.NeighborEvent{
			Type: abyss.ANDNeighborEventDebug,
			Text: "peer already connected"}
		return abyss.EPANIC
	}
	if _, ok := a.dead_peers[peer_hash]; ok {
		a.eventCh <- abyss.NeighborEvent{
			Type: abyss.ANDNeighborEventDebug,
			Text: "dead peer reconnected without cleanup"}
		return abyss.EPANIC
	}
	if a.local_hash == peer_hash {
		a.eventCh <- abyss.NeighborEvent{
			Type: abyss.ANDNeighborEventDebug,
			Text: "self connection"}
		return abyss.EPANIC
	}

	a.peers[peer_hash] = peer

	if DEBUG {
		for _, join_target := range a.join_targets {
			if _, ok := join_target.pre_members[peer_hash]; ok {
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "join target corrupted - PeerConnected"}
				return abyss.EPANIC
			}
		}
	}

	for lssid, join_aurl := range a.join_targets_connecting {
		delete(a.join_targets_connecting, lssid)
		if join_aurl.Hash == peer_hash {
			a.join_targets[lssid] = NewJoinTarget(peer, join_aurl.Path)
			if !peer.TrySendJN(lssid, join_aurl.Path) {
				a.dropPeer(peer)
			}
		}
	}

	for lssid, world := range a.worlds {
		if DEBUG {
			if _, ok := world.members[peer_hash]; ok {
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "world members corrupted - PeerConnected"}
				return abyss.EPANIC
			}
			if _, ok := world.snb_targets[peer_hash]; ok {
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "world snb_targets corrupted - PeerConnected"}
				return abyss.EPANIC
			}
			if _, ok := world.pre_members[peer_hash]; ok {
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "world pre_mem_members corrupted - PeerConnected"}
				return abyss.EPANIC
			}
		}
		if pre_conn_session_id, ok := world.pre_conn_members[peer_hash]; ok {
			delete(world.pre_conn_members, peer_hash)
			peer_session := abyss.ANDPeerSession{Peer: peer, PeerSessionID: pre_conn_session_id}
			world.pre_members[peer_hash] = PreMemSession{state: PRE_MEM_CONNECTED, ANDPeerSession: peer_session}
			a.eventCh <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: lssid,
				ANDPeerSession: peer_session,
			}
		}
	}
	return 0
}

func (a *AND) PeerClose(peer abyss.IANDPeer) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

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
		a.eventCh <- abyss.NeighborEvent{
			Type: abyss.ANDNeighborEventDebug,
			Text: "closing unknown peer: " + peer_hash}
		return abyss.EPANIC
	}
}

func (a *AND) OpenWorld(local_session_id uuid.UUID, world_url string) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	if _, ok := a.join_targets[local_session_id]; ok {
		return abyss.EINVAL
	}
	if _, ok := a.worlds[local_session_id]; ok {
		return abyss.EINVAL
	}

	a.worlds[local_session_id] = NewWorld(local_session_id, world_url)
	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinSuccess,
		LocalSessionID: local_session_id,
		Text:           world_url,
	}
	return 0
}

func (a *AND) JoinWorld(local_session_id uuid.UUID, abyss_url *aurl.AURL) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	peer, ok := a.peers[abyss_url.Hash]
	if !ok { //require connection
		//check for redundant call
		if _, ok := a.join_targets_connecting[local_session_id]; ok {
			return abyss.EINVAL
		}
		a.join_targets_connecting[local_session_id] = abyss_url
		a.eventCh <- abyss.NeighborEvent{
			Type:   abyss.ANDConnectRequest,
			Object: abyss_url,
		}
		return 0
	}

	if _, ok := a.join_targets[local_session_id]; ok {
		return abyss.EINVAL
	}
	if _, ok := a.worlds[local_session_id]; ok {
		return abyss.EINVAL
	}

	a.join_targets[local_session_id] = NewJoinTarget(peer, abyss_url.Path)
	if !peer.TrySendJN(local_session_id, abyss_url.Path) {
		a.dropPeer(peer)
	}
	return 0
}

// this does not guarantee join cancellation.
func (a *AND) CancelJoin(local_session_id uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//check for connecting join
	if _, ok := a.join_targets_connecting[local_session_id]; ok {
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDJoinFail,
			LocalSessionID: local_session_id,
			Text:           JNM_CANCELED,
			Value:          JNC_CANCELED,
		}
		delete(a.join_targets_connecting, local_session_id)
		return 0
	}

	join_target, ok := a.join_targets[local_session_id]
	if !ok {
		return abyss.EINVAL
	}
	dead_peers := make([]abyss.IANDPeer, 0, len(join_target.pre_members))
	for _, pre_mem := range join_target.pre_members {
		if !pre_mem.Peer.TrySendRST(local_session_id, pre_mem.PeerSessionID) {
			dead_peers = append(dead_peers, pre_mem.Peer)
		}
	}
	for _, dead_peer := range dead_peers {
		a.dropPeer(dead_peer)
	}

	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinFail,
		LocalSessionID: local_session_id,
		Text:           JNM_CANCELED,
		Value:          JNC_CANCELED,
	}
	delete(a.join_targets, local_session_id)
	return 0
}

func (a *AND) AcceptSession(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return abyss.EINVAL
	}

	peer_hash := peer_session.Peer.IDHash()
	pre_member, ok := world.pre_members[peer_hash]
	if !ok || pre_member.PeerSessionID != peer_session.PeerSessionID {
		return abyss.EINVAL
	}
	if DEBUG {
		if _, ok := world.members[peer_hash]; ok {
			a.eventCh <- abyss.NeighborEvent{
				Type: abyss.ANDNeighborEventDebug,
				Text: "world members corrupted - 3"}
			return abyss.EPANIC
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
		world.members[peer_hash] = pre_member.ANDPeerSession
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: local_session_id,
			ANDPeerSession: peer_session,
		}
		if !peer_session.Peer.TrySendMEM(local_session_id, peer_session.PeerSessionID) {
			a.dropPeer(peer_session.Peer)
			return 0
		}
		return 0
	case PRE_MEM_WAITING:
		return abyss.EINVAL
	case PRE_MEM_JN:
		pre_member.state = PRE_MEM_WAITING
		world.pre_members[peer_hash] = pre_member
		member_sessions := make([]abyss.ANDPeerSession, 0, len(world.members))
		dead_members := make([]abyss.IANDPeer, 0, len(world.members))
		for _, mem := range world.members {
			if !mem.Peer.TrySendJNI(world.local_session_id, mem.PeerSessionID, peer_session) {
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
		return abyss.EPANIC
	}
}

func (a *AND) DeclineSession(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, code int, message string) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return abyss.EINVAL
	}

	peer_hash := peer_session.Peer.IDHash()
	pre_mem, ok := world.pre_members[peer_hash]
	if !ok || pre_mem.PeerSessionID != peer_session.PeerSessionID {
		return abyss.EINVAL
	}

	switch pre_mem.state {
	case PRE_MEM_CONNECTED:
		delete(world.pre_members, peer_hash)
		return 0
	case PRE_MEM_RECVED:
		delete(world.pre_members, peer_hash)
		a.resetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return 0
	case PRE_MEM_WAITING:
		return abyss.EINVAL //cannot decline already accpted session
	case PRE_MEM_JN:
		delete(world.pre_members, peer_hash)
		if !peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, code, message) {
			a.dropPeer(peer_session.Peer)
		}
		return 0
	default:
		return abyss.EPANIC
	}
}

func (a *AND) ResetPeerSession(local_session_id uuid.UUID, peer abyss.IANDPeer, peer_session_id uuid.UUID) {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	a.resetOptDrop(peer, local_session_id, peer_session_id)
}

func (a *AND) CloseWorld(local_session_id uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return abyss.EINVAL
	}

	dead_peers := make([]abyss.IANDPeer, 0, len(world.members))
	for _, peer_session := range world.members {
		if !peer_session.Peer.TrySendRST(local_session_id, peer_session.PeerSessionID) {
			dead_peers = append(dead_peers, peer_session.Peer)
		}
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionClose,
			LocalSessionID: world.local_session_id,
			ANDPeerSession: peer_session,
		}
	}
	for _, dp := range dead_peers {
		a.dropPeer(dp)
	}

	delete(a.worlds, local_session_id)
	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDWorldLeave,
		LocalSessionID: world.local_session_id,
	}
	return 0
}

func (a *AND) TimerExpire(local_session_id uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//TODO: SNB
	return 0
}

// session_uuid is always the sender's session id.
func (a *AND) JN(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return abyss.EINVAL
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
	world.pre_members[peer_hash] = PreMemSession{state: PRE_MEM_JN, ANDPeerSession: peer_session}
	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDSessionRequest,
		LocalSessionID: local_session_id,
		ANDPeerSession: peer_session,
	}
	return 0
}
func (a *AND) JOK(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, world_url string, member_infos []abyss.ANDFullPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//check and clear join target
	join_target, ok := a.join_targets[local_session_id]
	if !ok {
		a.resetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return abyss.EINVAL
	}
	if join_target.peer != peer_session.Peer {
		a.resetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return abyss.EINVAL
	}
	delete(a.join_targets, local_session_id)

	//create world and add
	world := NewWorld(local_session_id, world_url)
	a.worlds[local_session_id] = world
	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinSuccess,
		LocalSessionID: local_session_id,
		Text:           world_url,
	}

	//append join target as member
	world.pre_members[peer_session.Peer.IDHash()] = PreMemSession{state: PRE_MEM_RECVED, ANDPeerSession: peer_session}
	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDSessionRequest,
		LocalSessionID: local_session_id,
		ANDPeerSession: peer_session,
	}

	//handle connected, MEM received-pre_member
	for pre_mem_hash, pre_mem_session := range join_target.pre_members {
		world.pre_members[pre_mem_hash] = PreMemSession{state: PRE_MEM_RECVED, ANDPeerSession: pre_mem_session}
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: local_session_id,
			ANDPeerSession: pre_mem_session,
		}
	}

	//handle non-connected/MEM not received members
	for _, member_info := range member_infos {
		if _, ok := world.pre_members[member_info.AURL.Hash]; ok {
			continue
		}
		if peer, ok := a.peers[member_info.AURL.Hash]; ok {
			//connected, MEM not received
			peer_session := abyss.ANDPeerSession{Peer: peer, PeerSessionID: member_info.SessionID}
			world.pre_members[member_info.AURL.Hash] = PreMemSession{
				state:          PRE_MEM_CONNECTED,
				ANDPeerSession: peer_session,
			}
			a.eventCh <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: local_session_id,
				ANDPeerSession: peer_session,
			}
		} else {
			//non-connected
			world.pre_conn_members[member_info.AURL.Hash] = member_info.SessionID
			a.eventCh <- abyss.NeighborEvent{
				Type: abyss.ANDPeerRegister,
				Object: &abyss.PeerCertificates{
					RootCertDer:         member_info.RootCertificateDer,
					HandshakeKeyCertDer: member_info.HandshakeKeyCertificateDer,
				},
			}
			a.eventCh <- abyss.NeighborEvent{
				Type:   abyss.ANDConnectRequest,
				Object: member_info.AURL,
			}
		}
	}

	return 0
}
func (a *AND) JDN(local_session_id uuid.UUID, peer abyss.IANDPeer, code int, message string) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	join_target, ok := a.join_targets[local_session_id]
	if !ok {
		return abyss.EINVAL
	}
	if join_target.peer != peer {
		return abyss.EINVAL
	}

	delete(a.join_targets, local_session_id) // it should never have pre members - what if it does? - IDK :/
	a.eventCh <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinFail,
		LocalSessionID: local_session_id,
		Text:           message,
		Value:          code,
	}
	return 0
}
func (a *AND) JNI(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, member_info abyss.ANDFullPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//member_hash string, member_aurl string, member_session_id uuid.UUID
	world, ok := a.worlds[local_session_id]
	if !ok {
		return abyss.EINVAL
	}
	if _, ok := world.members[peer_session.Peer.IDHash()]; !ok {
		a.resetOptDrop(peer_session.Peer, local_session_id, peer_session.PeerSessionID)
		return abyss.EINVAL
	}

	// this is vulnerable to fabricated JNI flooding
	if _, ok := world.members[member_info.AURL.Hash]; ok {
		return 0
	}
	if _, ok := world.pre_members[member_info.AURL.Hash]; ok {
		return 0
	}
	if _, ok := world.pre_conn_members[member_info.AURL.Hash]; ok {
		return 0
	}

	if member, ok := a.peers[member_info.AURL.Hash]; ok {
		member_session := abyss.ANDPeerSession{Peer: member, PeerSessionID: member_info.SessionID}
		world.pre_members[member_info.AURL.Hash] = PreMemSession{
			state:          PRE_MEM_CONNECTED,
			ANDPeerSession: member_session,
		}
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: local_session_id,
			ANDPeerSession: member_session,
		}
		return 0
	}

	world.pre_conn_members[member_info.AURL.Hash] = member_info.SessionID
	a.eventCh <- abyss.NeighborEvent{
		Type: abyss.ANDPeerRegister,
		Object: &abyss.PeerCertificates{
			RootCertDer:         member_info.RootCertificateDer,
			HandshakeKeyCertDer: member_info.HandshakeKeyCertificateDer,
		},
	}
	a.eventCh <- abyss.NeighborEvent{
		Type:   abyss.ANDConnectRequest,
		Object: member_info.AURL,
	}
	return 0
}
func (a *AND) MEM(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	peer_hash := peer_session.Peer.IDHash()

	world, ok := a.worlds[local_session_id]
	if !ok {
		// not world-related. search for join targets
		join_target, ok := a.join_targets[local_session_id]
		if !ok {
			// not join target either. ignore
			return abyss.EINVAL
		}

		if old_session, ok := join_target.pre_members[peer_hash]; ok {
			if old_session.PeerSessionID != peer_session.PeerSessionID {
				if !a.resetOptDrop(old_session.Peer, local_session_id, old_session.PeerSessionID) {
					return 0
				}
			}
			a.resetOptDrop(old_session.Peer, local_session_id, peer_session.PeerSessionID)
			return 0
		}

		// add pre_member in join target
		join_target.pre_members[peer_hash] = peer_session
		return 0
	}

	if old_session, ok := world.members[peer_hash]; ok {
		a.dropWorldMember(world, peer_hash)

		if old_session.PeerSessionID != peer_session.PeerSessionID {
			if !a.resetOptDrop(old_session.Peer, local_session_id, old_session.PeerSessionID) {
				return 0
			}
		}
		a.resetOptDrop(old_session.Peer, local_session_id, peer_session.PeerSessionID)
		return 0
	}

	if pre_mem, ok := world.pre_members[peer_hash]; ok {
		switch pre_mem.state {
		case PRE_MEM_CONNECTED:
			if pre_mem.PeerSessionID != peer_session.PeerSessionID {
				// peer session mismatch with expection
				delete(world.pre_members, peer_hash)
				if !a.resetOptDrop(pre_mem.Peer, local_session_id, pre_mem.PeerSessionID) {
					return 0
				}
				a.resetOptDrop(pre_mem.Peer, local_session_id, peer_session.PeerSessionID)
				return 0
			}
			// expected MEM
			pre_mem.state = PRE_MEM_RECVED
			world.pre_members[peer_hash] = pre_mem

		case PRE_MEM_WAITING:
			if pre_mem.PeerSessionID != peer_session.PeerSessionID {
				// peer session mismatch with expection
				delete(world.pre_members, peer_hash)
				if !a.resetOptDrop(pre_mem.Peer, local_session_id, pre_mem.PeerSessionID) {
					return 0
				}
				a.resetOptDrop(pre_mem.Peer, local_session_id, peer_session.PeerSessionID)
				return 0
			}
			// expected MEM
			delete(world.pre_members, peer_hash)
			world.members[peer_hash] = pre_mem.ANDPeerSession
			a.eventCh <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionReady,
				LocalSessionID: local_session_id,
				ANDPeerSession: pre_mem.ANDPeerSession,
			}

		case PRE_MEM_RECVED, PRE_MEM_JN:
			// 1) duplicate MEM - reset both

			// 2) MEM after JN
			// A -- C, B -JN> C, A -JN> B, C -JNI(B)> A, A -MEM> B. B has not accepted yet.
			// resetting both is most clean

			delete(world.pre_members, peer_hash)
			if pre_mem.PeerSessionID != peer_session.PeerSessionID {
				if !a.resetOptDrop(pre_mem.Peer, local_session_id, pre_mem.PeerSessionID) {
					return 0
				}
			}
			a.resetOptDrop(pre_mem.Peer, local_session_id, peer_session.PeerSessionID)
		}
		return 0
	}

	world.pre_members[peer_hash] = PreMemSession{state: PRE_MEM_RECVED, ANDPeerSession: peer_session}
	return 0
}

// TODO
func (a *AND) SNB(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	return 0
}
func (a *AND) CRR(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	return 0
}
func (a *AND) RST(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	if peer_session.PeerSessionID == uuid.Nil { //reset all for target local session, peer session wildcard
		if world, ok := a.worlds[local_session_id]; ok {
			a.dropWorldMember(world, peer_session.Peer.IDHash())
			delete(world.pre_members, peer_session.Peer.IDHash())
		} else if _, ok := a.join_targets[local_session_id]; ok {
			delete(a.join_targets, local_session_id)
			a.eventCh <- abyss.NeighborEvent{
				Type:           abyss.ANDJoinFail,
				LocalSessionID: local_session_id,
				Text:           JNM_RESET,
				Value:          JNC_RESET,
			}
		}
		return 0
	}

	//peer session targeted reset
	if world, ok := a.worlds[local_session_id]; ok {
		if member, ok := world.members[peer_session.Peer.IDHash()]; ok {
			if member.PeerSessionID == peer_session.PeerSessionID {
				a.dropWorldMember(world, peer_session.Peer.IDHash())
			}
		} else if pre_mem, ok := world.pre_members[peer_session.Peer.IDHash()]; ok {
			if pre_mem.PeerSessionID == peer_session.PeerSessionID {
				delete(world.pre_members, peer_session.Peer.IDHash())
			}
		} else if _, ok := a.join_targets[local_session_id]; ok { //join targets do not check local_session_id.
			delete(a.join_targets, local_session_id)
			a.eventCh <- abyss.NeighborEvent{
				Type:           abyss.ANDJoinFail,
				LocalSessionID: local_session_id,
				Text:           JNM_RESET,
				Value:          JNC_RESET,
			}
		}
	}

	return 0
}

func (a *AND) SOA(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, objects []abyss.ObjectInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//filter out invalid messages
	if world, ok := a.worlds[local_session_id]; ok {
		if member, ok := world.members[peer_session.Peer.IDHash()]; ok {
			if member.PeerSessionID == peer_session.PeerSessionID {
				a.eventCh <- abyss.NeighborEvent{
					Type:           abyss.ANDObjectAppend,
					LocalSessionID: local_session_id,
					ANDPeerSession: peer_session,
					Object:         objects,
				}
				return 0
			}
		}
	}

	return abyss.EINVAL
}
func (a *AND) SOD(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, objectIDs []uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//filter out invalid messages
	if world, ok := a.worlds[local_session_id]; ok {
		if member, ok := world.members[peer_session.Peer.IDHash()]; ok {
			if member.PeerSessionID == peer_session.PeerSessionID {
				a.eventCh <- abyss.NeighborEvent{
					Type:           abyss.ANDObjectDelete,
					LocalSessionID: local_session_id,
					ANDPeerSession: peer_session,
					Object:         objectIDs,
				}
				return 0
			}
		}
	}

	return abyss.EINVAL
}

func (a *AND) resetOptDrop(peer abyss.IANDPeer, local_session_id uuid.UUID, peer_session_id uuid.UUID) bool {
	if !peer.TrySendRST(local_session_id, peer_session_id) {
		a.dropPeer(peer)
		return false
	}
	return true
}

func (a *AND) dropWorldMember(world *World, peer_hash string) {
	if peer_session, ok := world.members[peer_hash]; ok {
		delete(world.members, peer_hash)
		delete(world.snb_targets, peer_hash)
		if DEBUG {
			if _, ok := world.pre_members[peer_hash]; ok {
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "world pre_MEM_members corrupted - dropWorldMember",
				}
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "world pre_MEM_members corrupted - dropWorldMember",
				}
			}
			if _, ok := world.pre_conn_members[peer_hash]; ok {
				a.eventCh <- abyss.NeighborEvent{
					Type: abyss.ANDNeighborEventDebug,
					Text: "world pre_conn_members corrupted - dropWorldMember",
				}
			}
		}
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionClose,
			LocalSessionID: world.local_session_id,
			ANDPeerSession: peer_session,
		}
	}

	if DEBUG {
		if _, ok := world.snb_targets[peer_hash]; ok {
			a.eventCh <- abyss.NeighborEvent{
				Type: abyss.ANDNeighborEventDebug,
				Text: "world snb_targets corrupted - dropWorldMember"}
		}
	}

	delete(world.pre_members, peer_hash)
	delete(world.pre_conn_members, peer_hash)
}

func (a *AND) dropPeer(peer abyss.IANDPeer) {
	peer_hash := peer.IDHash()
	if DEBUG {
		if _, ok := a.peers[peer_hash]; !ok {
			a.eventCh <- abyss.NeighborEvent{
				Type: abyss.ANDNeighborEventDebug,
				Text: "peer not found - dropPeer"}
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
			pre_mem.Peer.TrySendRST(uuid.Nil, pre_mem.PeerSessionID)
			// we don't handle its failure for now;
			// it is better to be handled, only if we can clearly predict the recursion
			// but practically, no
		}
		delete(a.join_targets, ls)
		a.eventCh <- abyss.NeighborEvent{
			Type:           abyss.ANDJoinFail,
			LocalSessionID: ls,
			Text:           JNM_CLOSED,
			Value:          JNC_CLOSED,
		}
	}
	for _, world := range a.worlds {
		a.dropWorldMember(world, peer_hash)
	}
}

func (a *AND) CheckSanity() error {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	//check self connection
	if _, ok := a.peers[a.local_hash]; ok {
		return errors.New("self connection")
	}
	if _, ok := a.dead_peers[a.local_hash]; ok {
		return errors.New("self connection (dead)")
	}

	//ensure peers-dead_peers exclusivity
	for name := range a.peers {
		if _, ok := a.dead_peers[name]; ok {
			return errors.New("peer dead and alive (1)")
		}
	}
	for name := range a.dead_peers {
		if _, ok := a.peers[name]; ok {
			return errors.New("peer dead and alive (2)")
		}
	}

	//join_connecting - connection state
	for _, url := range a.join_targets_connecting {
		if _, ok := a.peers[url.Hash]; ok {
			return errors.New("connecting connected peer (join)")
		}
		if _, ok := a.dead_peers[url.Hash]; ok {
			return errors.New("connecting dead peer (join)")
		}
	}

	//join connected - connection state
	for _, targ := range a.join_targets {
		if _, ok := a.peers[targ.peer.IDHash()]; !ok {
			return errors.New("join target not connected")
		}
		if _, ok := a.dead_peers[targ.peer.IDHash()]; ok {
			return errors.New("join target dead")
		}
	}

	//world peers connetion state
	for _, world := range a.worlds {
		for _, mem := range world.members {
			if _, ok := a.peers[mem.Peer.IDHash()]; !ok {
				return errors.New("member not connected")
			}
			if _, ok := a.dead_peers[mem.Peer.IDHash()]; ok {
				return errors.New("member dead")
			}
		}
		for _, cmem := range world.pre_members {
			if _, ok := a.peers[cmem.Peer.IDHash()]; !ok {
				return errors.New("pre_member not connected")
			}
			if _, ok := a.dead_peers[cmem.Peer.IDHash()]; ok {
				return errors.New("pre_member dead")
			}
		}
		for prec := range world.pre_conn_members {
			if _, ok := a.peers[prec]; ok {
				return errors.New("pre_conn_member already connected")
			}
			if _, ok := a.dead_peers[prec]; ok {
				return errors.New("pre_conn_member dead")
			}
		}
	}

	//world state exclusivity
	for id := range a.join_targets_connecting {
		if _, ok := a.join_targets[id]; ok {
			return errors.New("world corrupted (1)")
		}
		if _, ok := a.worlds[id]; ok {
			return errors.New("world corrupted (2)")
		}
	}
	for id := range a.join_targets {
		if _, ok := a.join_targets_connecting[id]; ok {
			return errors.New("world corrupted (3)")
		}
		if _, ok := a.worlds[id]; ok {
			return errors.New("world corrupted (4)")
		}
	}
	for id := range a.worlds {
		if _, ok := a.join_targets_connecting[id]; ok {
			return errors.New("world corrupted (5)")
		}
		if _, ok := a.join_targets[id]; ok {
			return errors.New("world corrupted (6)")
		}
	}

	return nil
}
