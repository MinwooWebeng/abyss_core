package net_service

import (
	"fmt"

	"github.com/google/uuid"

	"abyss_neighbor_discovery/ahmp"
	aurl "abyss_neighbor_discovery/aurl"
	abyss "abyss_neighbor_discovery/interfaces"
	"abyss_neighbor_discovery/tools/functional"
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

func (p *AbyssPeer) AURL() *aurl.AURL {
	return &aurl.AURL{
		Scheme:    "abyss",
		Hash:      p.inbound.identity,
		Addresses: p.outbound.addresses,
		Path:      "/",
	}
}

func (p *AbyssPeer) AhmpCh() chan any {
	return p.inbound.AhmpChannel
}

func (p *AbyssPeer) TrySendJN(local_session_id uuid.UUID, path string) bool {
	fmt.Println("sending JN")
	if p.outbound.cbor_encoder.Encode(ahmp.JN_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawJN{
		SenderSessionID: local_session_id.String(),
		Text:            path,
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendJOK(peer_session_id uuid.UUID, local_session_id uuid.UUID, world_url string, member_sessions []abyss.ANDPeerSession) bool {
	fmt.Println("sending JOK")
	if p.outbound.cbor_encoder.Encode(ahmp.JOK_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawJOK{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
		Neighbors: functional.Filter(member_sessions, func(session abyss.ANDPeerSession) ahmp.SessionInfoText {
			return ahmp.SessionInfoText{
				AURL:      session.Peer.AURL().ToString(),
				SessionID: session.PeerSessionID.String(),
			}
		}),
		Text: world_url,
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool {
	fmt.Println("sending JDN")
	if p.outbound.cbor_encoder.Encode(ahmp.JDN_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawJDN{
		RecverSessionID: peer_session_id.String(),
		Text:            message,
		Code:            code,
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendJNI(peer_session_id uuid.UUID, local_session_id uuid.UUID, member_session abyss.ANDPeerSession) bool {
	fmt.Println("sending JNI")
	if p.outbound.cbor_encoder.Encode(ahmp.JNI_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawJNI{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
		Neighbor: ahmp.SessionInfoText{
			AURL:      member_session.Peer.AURL().ToString(),
			SessionID: member_session.PeerSessionID.String(),
		},
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendMEM(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool {
	fmt.Println("sending MEM")
	if p.outbound.cbor_encoder.Encode(ahmp.MEM_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawMEM{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendSNB(peer_session_id uuid.UUID, local_session_id uuid.UUID, member_sessions []abyss.ANDPeerSessionInfo) bool {
	if p.outbound.cbor_encoder.Encode(ahmp.SNB_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawSNB{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendCRR(peer_session_id uuid.UUID, local_session_id uuid.UUID, member_sessions []abyss.ANDPeerSessionInfo) bool {
	if p.outbound.cbor_encoder.Encode(ahmp.CRR_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawCRR{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendRST(peer_session_id uuid.UUID, local_session_id uuid.UUID) bool {
	if p.outbound.cbor_encoder.Encode(ahmp.RST_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawRST{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}) != nil {
		return false
	}
	return true
}

func (p *AbyssPeer) TrySendSOA(peer_session_id uuid.UUID, local_session_id uuid.UUID, objects []abyss.ObjectInfo) bool {
	if p.outbound.cbor_encoder.Encode(ahmp.SOA_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawSOA{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}) != nil {
		return false
	}
	return true
}
func (p *AbyssPeer) TrySendSOD(peer_session_id uuid.UUID, local_session_id uuid.UUID, objectIDs []uuid.UUID) bool {
	if p.outbound.cbor_encoder.Encode(ahmp.SOD_T) != nil {
		return false
	}
	if p.outbound.cbor_encoder.Encode(ahmp.RawSOD{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}) != nil {
		return false
	}
	return true
}
