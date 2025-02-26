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

func (w *World) GetEventChannel() chan any {
	return w.eventChannel
}
func (w *World) Leave() {
	if w.origin.CloseWorld(w.session_id) != 0 {
		panic("World Leave failed")
	}
}

func (w *World) RaisePeerRequest(peer_session abyss.PeerSession) {
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
func (w *World) RaisePeerReady(peer_session abyss.PeerSession) {
	w.eventChannel <- abyss.EWorldPeerReady{
		Peer: &Peer{
			world: w,
			hash:  peer_session.Peer.IDHash(),
		},
	}
}
