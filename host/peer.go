package host

import (
	abyss "abyss_neighbor_discovery/interfaces"

	"github.com/google/uuid"
)

type Peer struct {
	world       *World
	hash        string
	peerSession abyss.PeerSession
}

func (p *Peer) Hash() string {
	return p.hash
}
func (p *Peer) AppendObjects(objects []abyss.ObjectInfo) {
	p.world.origin.AppendObject(p.world.session_id, p.peerSession, objects)
}
func (p *Peer) DeleteObjects(objectIDs []uuid.UUID) {
	p.world.origin.DeleteObject(p.world.session_id, p.peerSession, objectIDs)
}
func (p *Peer) Close() {
	p.world.origin.ConfirmLeave(p.world.session_id, p.peerSession)
}
