package host

import (
	abyss "abyss_core/interfaces"

	"github.com/google/uuid"
)

type Peer struct {
	world       *World
	hash        string
	peerSession abyss.ANDPeerSession
}

func (p *Peer) Hash() string {
	return p.hash
}
func (p *Peer) AppendObjects(objects []abyss.ObjectInfo) bool {
	return p.peerSession.Peer.TrySendSOA(p.world.session_id, p.peerSession.PeerSessionID, objects)
}
func (p *Peer) DeleteObjects(objectIDs []uuid.UUID) bool {
	return p.peerSession.Peer.TrySendSOD(p.world.session_id, p.peerSession.PeerSessionID, objectIDs)
}
