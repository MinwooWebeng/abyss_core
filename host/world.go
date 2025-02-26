package host

import (
	abyss "abyss_neighbor_discovery/interfaces"

	"github.com/google/uuid"
)

type World struct {
	origin       abyss.INeighborDiscovery
	session_id   uuid.UUID
	eventChannel chan abyss.WorldEvents
}

func (w *World) GetEventChannel() chan abyss.WorldEvents {
	return w.eventChannel
}
func (w *World) ApproveSessionRequest(peer_session abyss.PeerSession) {
	w.origin.AcceptSession(w.session_id, peer_session)
}
func (w *World) DeclineSessionRequest(peer_session abyss.PeerSession, code int, message string) {
	w.origin.DeclineSession(w.session_id, peer_session, code, message)
}
func (w *World) AppendObjectsToPeer(peer_hash string, objects []abyss.ObjectInfo) {

}
func (w *World) DeleteObjectsToPeer(peer_hash string, objects []int) {

}
func (w *World) Leave() {
	if w.origin.CloseWorld(w.session_id) != 0 {
		panic("World Leave failed")
	}
}
