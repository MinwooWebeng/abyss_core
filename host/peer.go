package host

import (
	abyss "abyss_neighbor_discovery/interfaces"

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
	return p.peerSession.Peer.TrySendSOA(p.peerSession.PeerSessionID, p.world.session_id, objects)
}
func (p *Peer) DeleteObjects(objectIDs []uuid.UUID) bool {
	return p.peerSession.Peer.TrySendSOD(p.peerSession.PeerSessionID, p.world.session_id, objectIDs)
}
func (p *Peer) Close() {
	p.world.origin.ConfirmLeave(p.world.session_id, p.peerSession)
}
