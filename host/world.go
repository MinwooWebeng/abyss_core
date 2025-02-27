package host

import (
	abyss "abyss_neighbor_discovery/interfaces"

	"github.com/google/uuid"
)

type World struct {
	origin       abyss.INeighborDiscovery
	session_id   uuid.UUID
	eventChannel chan any
}

func NewWorld(origin abyss.INeighborDiscovery, session_id uuid.UUID) *World {
	return &World{
		origin:       origin,
		session_id:   session_id,
		eventChannel: make(chan any, 16),
	}
}

func (w *World) SessionID() uuid.UUID { return w.session_id }
func (w *World) GetEventChannel() chan any {
	return w.eventChannel
}

func (w *World) RaisePeerRequest(peer_session abyss.ANDPeerSession) {
	w.eventChannel <- abyss.EWorldPeerRequest{
		PeerHash: peer_session.Peer.IDHash(),
		Accept: func() {
			w.origin.AcceptSession(w.session_id, peer_session)
		},
		Decline: func(code int, message string) {
			w.origin.DeclineSession(w.session_id, peer_session, code, message)
		},
	}
}
func (w *World) RaisePeerReady(peer_session abyss.ANDPeerSession) {
	w.eventChannel <- abyss.EWorldPeerReady{
		Peer: &Peer{
			world: w,
			hash:  peer_session.Peer.IDHash(),
		},
	}
}
func (w *World) RaiseObjectAppend(peer_hash string, objects []abyss.ObjectInfo) {
	w.eventChannel <- abyss.EPeerObjectAppend{
		PeerHash: peer_hash,
		Objects:  objects,
	}
}
func (w *World) RaiseObjectDelete(peer_hash string, objectIDs []uuid.UUID) {
	w.eventChannel <- abyss.EPeerObjectDelete{
		PeerHash:  peer_hash,
		ObjectIDs: objectIDs,
	}
}
func (w *World) RaisePeerLeave(peer_hash string) {
	w.eventChannel <- abyss.EWorldPeerLeave{
		PeerHash: peer_hash,
	}
}
