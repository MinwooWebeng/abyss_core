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
	origin    *BetaNetService
	peer_hash string
	inbound   AbyssInbound
	outbound  AbyssOutbound
}

func NewAbyssPeer(origin *BetaNetService, inbound AbyssInbound, outbound AbyssOutbound) *AbyssPeer {
	result := new(AbyssPeer)
	result.origin = origin
	result.peer_hash = outbound.peer_hash
	result.inbound = inbound
	result.outbound = outbound
	return result
}

func (p *AbyssPeer) IDHash() string {
	return p.peer_hash
}
func (p *AbyssPeer) AURL() *aurl.AURL {
	return &aurl.AURL{
		Scheme:    "abyss",
		Hash:      p.inbound.peer_hash,
		Addresses: p.outbound.addresses,
		Path:      "/",
	}
}

func (p *AbyssPeer) AhmpCh() chan any {
	return p.inbound.AhmpChannel
}

func (p *AbyssPeer) TrySendJN(local_session_id uuid.UUID, path string) bool {
	fmt.Println(struct {
		t string
		a any
	}{t: p.origin.localIdentity.IDHash() + ">" + p.peer_hash + " sending JN", a: ahmp.RawJN{
		SenderSessionID: local_session_id.String(),
		Text:            path,
	}})
	if p.outbound.cbor_encoder.Encode(ahmp.JN_T) != nil {
		return false
	}
	return p.outbound.cbor_encoder.Encode(ahmp.RawJN{
		SenderSessionID: local_session_id.String(),
		Text:            path,
	}) == nil
}
func (p *AbyssPeer) TrySendJOK(local_session_id uuid.UUID, peer_session_id uuid.UUID, world_url string, member_sessions []abyss.ANDPeerSession) bool {
	fmt.Println(struct {
		t string
		a any
	}{t: p.origin.localIdentity.IDHash() + ">" + p.peer_hash + " sending JOK", a: ahmp.RawJOK{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
		Neighbors: functional.Filter(member_sessions, func(session abyss.ANDPeerSession) ahmp.SessionInfoText {
			return ahmp.SessionInfoText{
				AURL:      session.Peer.AURL().ToString(),
				SessionID: session.PeerSessionID.String(),
			}
		}),
		Text: world_url,
	}})
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
	fmt.Println(struct {
		t string
		a any
	}{t: p.origin.localIdentity.IDHash() + ">" + p.peer_hash + " sending JDN", a: ahmp.RawJDN{
		RecverSessionID: peer_session_id.String(),
		Text:            message,
		Code:            code,
	}})
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
func (p *AbyssPeer) TrySendJNI(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_session abyss.ANDPeerSession) bool {
	fmt.Println(struct {
		t string
		a any
	}{t: p.origin.localIdentity.IDHash() + ">" + p.peer_hash + " sending JNI", a: ahmp.RawJNI{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
		Neighbor: ahmp.SessionInfoText{
			AURL:      member_session.Peer.AURL().ToString(),
			SessionID: member_session.PeerSessionID.String(),
		},
	}})
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
func (p *AbyssPeer) TrySendMEM(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool {
	fmt.Println(struct {
		t string
		a any
	}{t: p.origin.localIdentity.IDHash() + ">" + p.peer_hash + " sending MEM", a: ahmp.RawMEM{
		SenderSessionID: local_session_id.String(),
		RecverSessionID: peer_session_id.String(),
	}})
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
func (p *AbyssPeer) TrySendSNB(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_sessions []abyss.ANDPeerSessionInfo) bool {
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
func (p *AbyssPeer) TrySendCRR(local_session_id uuid.UUID, peer_session_id uuid.UUID, member_sessions []abyss.ANDPeerSessionInfo) bool {
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
func (p *AbyssPeer) TrySendRST(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool {
	fmt.Println(struct {
		t string
		a any
	}{t: p.origin.localIdentity.IDHash() + ">" + p.peer_hash + " sending RST", a: ahmp.RawRST{
		RecverSessionID: peer_session_id.String(),
	}})
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

func (p *AbyssPeer) TrySendSOA(local_session_id uuid.UUID, peer_session_id uuid.UUID, objects []abyss.ObjectInfo) bool {
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
func (p *AbyssPeer) TrySendSOD(local_session_id uuid.UUID, peer_session_id uuid.UUID, objectIDs []uuid.UUID) bool {
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
