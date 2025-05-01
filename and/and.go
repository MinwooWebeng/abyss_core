package and

import (
	"sync"

	"github.com/google/uuid"

	"github.com/MinwooWebeng/abyss_core/aurl"
	abyss "github.com/MinwooWebeng/abyss_core/interfaces"
)

type AND struct {
	eventCh chan abyss.NeighborEvent

	local_hash string

	peers  map[string]abyss.IANDPeer //id hash - peer
	worlds map[uuid.UUID]*ANDWorld   //local session id - world

	api_mtx *sync.Mutex
}

func NewAND(local_hash string) *AND {
	return &AND{
		eventCh:    make(chan abyss.NeighborEvent, 64),
		local_hash: local_hash,
		peers:      make(map[string]abyss.IANDPeer),
		worlds:     make(map[uuid.UUID]*ANDWorld),
		api_mtx:    new(sync.Mutex),
	}
}

func (a *AND) EventChannel() chan abyss.NeighborEvent {
	return a.eventCh
}

func (a *AND) PeerConnected(peer abyss.IANDPeer) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	a.peers[peer.IDHash()] = peer

	for _, world := range a.worlds {
		world.PeerConnected(peer)
	}
	return 0
}

func (a *AND) PeerClose(peer abyss.IANDPeer) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	for _, world := range a.worlds {
		world.RemovePeer(peer)
	}
	delete(a.peers, peer.IDHash())
	return 0
}

func (a *AND) OpenWorld(local_session_id uuid.UUID, world_url string) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world := NewWorldOpen(local_session_id, world_url, a.peers, a.eventCh)
	a.worlds[world.lsid] = world
	return 0
}

func (a *AND) JoinWorld(local_session_id uuid.UUID, abyss_url *aurl.AURL) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world := NewWorldJoin(local_session_id, abyss_url, a.peers, a.eventCh)
	a.worlds[world.lsid] = world
	return 0
}

func (a *AND) AcceptSession(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.AcceptSession(peer_session)
	return 0
}

func (a *AND) DeclineSession(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, code int, message string) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.DeclineSession(peer_session, code, message)
	return 0
}

func (a *AND) CloseWorld(local_session_id uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.Close()
	return 0
}

func (a *AND) TimerExpire(local_session_id uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.TimerExpire()
	return 0
}

// session_uuid is always the sender's session id.
func (a *AND) JN(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.JN(peer_session)
	return 0
}
func (a *AND) JOK(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, world_url string, member_infos []abyss.ANDFullPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.JOK(peer_session, world_url, member_infos)
	return 0
}
func (a *AND) JDN(local_session_id uuid.UUID, peer abyss.IANDPeer, code int, message string) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.Close()
	return 0
}
func (a *AND) JNI(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, member_info abyss.ANDFullPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.JNI(peer_session, member_info)
	return 0
}
func (a *AND) MEM(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.MEM(peer_session)
	return 0
}
func (a *AND) SJN(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.SJN(peer_session, member_infos)
	return 0
}
func (a *AND) CRR(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.CRR(peer_session, member_infos)
	return 0
}
func (a *AND) RST(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	if local_session_id != uuid.Nil {
		world, ok := a.worlds[local_session_id]
		if !ok {
			return 0
		}
		world.RST(peer_session)
	} else {
		for _, world := range a.worlds {
			world.RST(peer_session)
		}
	}
	return 0
}

func (a *AND) SOA(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, objects []abyss.ObjectInfo) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.SOA(peer_session, objects)
	return 0
}
func (a *AND) SOD(local_session_id uuid.UUID, peer_session abyss.ANDPeerSession, objectIDs []uuid.UUID) abyss.ANDERROR {
	a.api_mtx.Lock()
	defer a.api_mtx.Unlock()

	world, ok := a.worlds[local_session_id]
	if !ok {
		return 0
	}
	world.SOD(peer_session, objectIDs)
	return 0
}
