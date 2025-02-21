package net_service

import (
	"github.com/google/uuid"

	abyss "abyss_neighbor_discovery/interfaces"
)

type AbyssPeer struct {
	origin *BetaNetService
	abyss.IRemoteIdentity
	inbound  AbyssInbound
	outbound AbyssOutbound
}

func NewAbyssPeer(origin *BetaNetService, inbound AbyssInbound, outbound AbyssOutbound) *AbyssPeer {
	result := new(AbyssPeer)
	result.origin = origin
	result.IRemoteIdentity = outbound.identity
	result.inbound = inbound
	result.outbound = outbound
	return result
}

func (p *AbyssPeer) AhmpCh() chan abyss.IAhmpMessage {
	return nil
}

func (p *AbyssPeer) TrySendJN(local_session_id uuid.UUID, path string) bool {
	return false
}
func (p *AbyssPeer) TrySendJOK(peer_session_id uuid.UUID, local_session_id uuid.UUID, world_url string, member_sessions []abyss.PeerSession) bool {
	return false
}
func (p *AbyssPeer) TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool {
	return false
}
func (p *AbyssPeer) TrySendJNI(peer_session_id uuid.UUID, member_session abyss.PeerSession) bool {
	return false
}
func (p *AbyssPeer) TrySendMEM(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool {
	return false
}
func (p *AbyssPeer) TrySendSNB(peer_session_id uuid.UUID, member_sessions []abyss.PeerSessionInfo) bool {
	return false
}
func (p *AbyssPeer) TrySendCRR(peer_session_id uuid.UUID, member_sessions []abyss.PeerSessionInfo) bool {
	return false
}
func (p *AbyssPeer) TrySendRST(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool {
	return false
}

func (p *AbyssPeer) Close() {
	p.origin.confirmPeerShutdown(p)
}
