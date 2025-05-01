package and

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/MinwooWebeng/abyss_core/aurl"
	abyss "github.com/MinwooWebeng/abyss_core/interfaces"
)

const (
	WS_DC_JT     int = iota + 1 //disconnected join target
	WS_DC_JNI                   //disconnected, JNI received
	WS_CC                       //connected, no info
	WS_JT                       //JN sent
	WS_JN                       //JN received
	WS_RMEM_NJNI                //MEM received, JNI not received
	WS_JNI                      //JNI received
	WS_RMEM                     //MEM received
	WS_TMEM                     //MEM sent
	WS_MEM                      //member
)

type ANDPeerSessionInfo struct {
	s     abyss.ANDPeerSession
	state int
	t_mem time.Time
	sjnp  bool //is sjn suppressed
	sjnc  int  //sjn receive count
}

type ANDWorld struct {
	lsid  uuid.UUID
	path  string                         //join path
	wurl  string                         //world url
	peers map[string]*ANDPeerSessionInfo //key: hash

	ech       chan abyss.NeighborEvent
	is_closed bool
}

func NewWorldOpen(local_session_id uuid.UUID, world_url string, connected_members map[string]abyss.IANDPeer, event_ch chan abyss.NeighborEvent) *ANDWorld {
	result := &ANDWorld{
		lsid:  local_session_id,
		wurl:  world_url,
		peers: make(map[string]*ANDPeerSessionInfo),
		ech:   event_ch,
	}
	for _, peer := range connected_members {
		result.peers[peer.IDHash()] = &ANDPeerSessionInfo{
			s: abyss.ANDPeerSession{
				Peer:          peer,
				PeerSessionID: uuid.Nil,
			},
			state: WS_CC,
		}
	}
	result.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinSuccess,
		LocalSessionID: result.lsid,
		Text:           world_url,
	}
	result.ech <- abyss.NeighborEvent{
		Type:  abyss.ANDTimerRequest,
		Value: rand.Intn(300),
	}
	return result
}

func NewWorldJoin(local_session_id uuid.UUID, target *aurl.AURL, connected_members map[string]abyss.IANDPeer, event_ch chan abyss.NeighborEvent) *ANDWorld {
	result := &ANDWorld{
		lsid:  local_session_id,
		path:  target.Path,
		peers: make(map[string]*ANDPeerSessionInfo),
		ech:   event_ch,
	}
	for _, peer := range connected_members {
		result.peers[peer.IDHash()] = &ANDPeerSessionInfo{
			s: abyss.ANDPeerSession{
				Peer:          peer,
				PeerSessionID: uuid.Nil,
			},
			state: WS_CC,
		}
	}
	if _, ok := connected_members[target.Hash]; !ok {
		result.peers[target.Hash] = &ANDPeerSessionInfo{
			s: abyss.ANDPeerSession{
				Peer:          nil,
				PeerSessionID: uuid.Nil,
			},
			state: WS_DC_JT,
		}
		result.ech <- abyss.NeighborEvent{
			Type:   abyss.ANDConnectRequest,
			Object: target,
		}
	} else {
		info := result.peers[target.Hash]
		info.state = WS_JT
		info.s.Peer.TrySendJN(result.lsid, target.Path)
	}

	result.ech <- abyss.NeighborEvent{
		Type:  abyss.ANDTimerRequest,
		Value: rand.Intn(300),
	}
	return result
}

func (w *ANDWorld) PeerConnected(peer abyss.IANDPeer) {
	info, ok := w.peers[peer.IDHash()]
	if !ok {
		w.peers[peer.IDHash()] = &ANDPeerSessionInfo{
			s: abyss.ANDPeerSession{
				Peer:          peer,
				PeerSessionID: uuid.Nil,
			},
			state: WS_CC,
		}
		return
	}
	info.s.Peer = peer
	switch info.state {
	case WS_DC_JT:
		peer.TrySendJN(w.lsid, w.path)
		info.state = WS_JT
	case WS_DC_JNI:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.state = WS_JNI
	}
}
func (w *ANDWorld) JN(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	switch info.state {
	case WS_CC:
		info.s.PeerSessionID = peer_session.PeerSessionID
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.state = WS_JN
	default:
		peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_INVALID_STATES, JNM_INVALID_STATES)
		return
	}
}
func (w *ANDWorld) JOK(peer_session abyss.ANDPeerSession, world_url string, member_infos []abyss.ANDFullPeerSessionInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	switch info.state {
	case WS_JT:
		info.s.PeerSessionID = peer_session.PeerSessionID
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDJoinSuccess,
			LocalSessionID: w.lsid,
			Text:           world_url,
		}
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.state = WS_RMEM
		info.sjnp = true

		for _, mem_info := range member_infos {
			w.JNI_MEMS(mem_info)
		}
	default:
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
}
func (w *ANDWorld) JNI(peer_session abyss.ANDPeerSession, member_info abyss.ANDFullPeerSessionInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_MEM:
		w.JNI_MEMS(member_info)
	default:
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
	}
}
func (w *ANDWorld) JNI_MEMS(mem_info abyss.ANDFullPeerSessionInfo) {
	info, ok := w.peers[mem_info.AURL.Hash]
	if !ok {
		w.peers[mem_info.AURL.Hash] = &ANDPeerSessionInfo{
			s: abyss.ANDPeerSession{
				Peer:          nil,
				PeerSessionID: mem_info.SessionID,
			},
			state: WS_DC_JNI,
			sjnp:  true,
		}
		w.ech <- abyss.NeighborEvent{
			Type: abyss.ANDPeerRegister,
			Object: &abyss.PeerCertificates{
				RootCertDer:         mem_info.RootCertificateDer,
				HandshakeKeyCertDer: mem_info.HandshakeKeyCertificateDer,
			},
		}
		w.ech <- abyss.NeighborEvent{
			Type:   abyss.ANDConnectRequest,
			Object: mem_info.AURL,
		}
		return
	}
	switch info.state {
	case WS_CC:
		info.s.PeerSessionID = mem_info.SessionID
		info.state = WS_JNI
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
	case WS_JT:
		panic("two join target")
	case WS_JN:
	case WS_RMEM_NJNI:
		if info.s.PeerSessionID != mem_info.SessionID {
			return
		}
		info.state = WS_RMEM
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
	case WS_JNI:
	case WS_RMEM:
	case WS_TMEM:
	case WS_MEM:
	}
}
func (w *ANDWorld) MEM(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_CC:
		info.state = WS_RMEM_NJNI
	case WS_JT:
	case WS_JN:
	case WS_RMEM_NJNI:
	case WS_JNI:
		info.state = WS_RMEM
	case WS_RMEM:
	case WS_TMEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.t_mem = time.Now()
		info.state = WS_MEM
	case WS_MEM:
	}
}
func (w *ANDWorld) SJN(peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_MEM:
		for _, mem_info := range member_infos {
			w.SJN_MEMS(peer_session, mem_info)
		}
	default:
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
	}
}
func (w *ANDWorld) SJN_MEMS(origin abyss.ANDPeerSession, mem_info abyss.ANDPeerSessionInfo) {
	info := w.peers[mem_info.PeerHash]
	if info.s.PeerSessionID != mem_info.SessionID {
		return
	}
	switch info.state {
	case WS_MEM:
		info.sjnc++
	default:
		origin.Peer.TrySendCRR(w.lsid, origin.PeerSessionID, []abyss.ANDPeerSessionInfo{mem_info})
	}
}
func (w *ANDWorld) CRR(peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_MEM:
		for _, mem_info := range member_infos {
			w.CRR_MEMS(peer_session, mem_info)
		}
	default:
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
	}
}
func (w *ANDWorld) CRR_MEMS(origin abyss.ANDPeerSession, mem_info abyss.ANDPeerSessionInfo) {
	info := w.peers[mem_info.PeerHash]
	if info.s.PeerSessionID != mem_info.SessionID {
		return
	}
	switch info.state {
	case WS_MEM:
		origin.Peer.TrySendJNI(w.lsid, origin.PeerSessionID, info.s)
		info.s.Peer.TrySendJNI(w.lsid, info.s.PeerSessionID, origin)
	}
}
func (w *ANDWorld) SOA(peer_session abyss.ANDPeerSession, objects []abyss.ObjectInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	//fmt.Println(info.state)
	switch info.state {
	case WS_MEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDObjectAppend,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
			Object:         objects,
		}
	}
}
func (w *ANDWorld) SOD(peer_session abyss.ANDPeerSession, objectIDs []uuid.UUID) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_MEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDObjectDelete,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
			Object:         objectIDs,
		}
	}
}
func (w *ANDWorld) RST(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		return
	}
	switch info.state {
	case WS_CC:
	case WS_JT:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDJoinFail,
			LocalSessionID: w.lsid,
			Text:           JNM_CANCELED,
			Value:          JNC_CANCELED,
		}
	case WS_JN:
	case WS_RMEM_NJNI:
	case WS_JNI:
	case WS_RMEM:
	case WS_TMEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
		}
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionClose,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
		}
	case WS_MEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
		}
	}
	delete(w.peers, peer_session.Peer.IDHash())
}

func (w *ANDWorld) AcceptSession(peer_session abyss.ANDPeerSession) {
	info, ok := w.peers[peer_session.Peer.IDHash()]
	if !ok {
		return
	}
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	fmt.Println(info.state)
	switch info.state {
	case WS_CC:
	case WS_JT:
	case WS_JN:
		member_infos := make([]abyss.ANDPeerSession, 0)
		for _, p := range w.peers {
			if p.state != WS_MEM {
				continue
			}
			member_infos = append(member_infos, p.s)
			p.s.Peer.TrySendJNI(w.lsid, p.s.PeerSessionID, info.s)
		}
		info.s.Peer.TrySendJOK(w.lsid, info.s.PeerSessionID, w.wurl, member_infos)
		info.state = WS_TMEM
	case WS_RMEM_NJNI:
	case WS_JNI:
		info.s.Peer.TrySendMEM(w.lsid, info.s.PeerSessionID)
		info.state = WS_TMEM
	case WS_RMEM:
		info.s.Peer.TrySendMEM(w.lsid, info.s.PeerSessionID)
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.t_mem = time.Now()
		info.state = WS_MEM
	case WS_TMEM:
	case WS_MEM:
	}
}
func (w *ANDWorld) DeclineSession(peer_session abyss.ANDPeerSession, code int, message string) {
	info, ok := w.peers[peer_session.Peer.IDHash()]
	if !ok {
		return
	}
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_CC:
	case WS_JT:
	case WS_JN:
	case WS_RMEM_NJNI:
	case WS_JNI:
	case WS_RMEM:
	case WS_TMEM:
	case WS_MEM:
	}
}
func (w *ANDWorld) TimerExpire() {
	sjn_mem := make([]abyss.ANDPeerSessionInfo, 0)
	for _, info := range w.peers {
		if info.state != WS_MEM ||
			time.Since(info.t_mem) < time.Second ||
			info.sjnp || info.sjnc > 3 {
			continue
		}
		sjn_mem = append(sjn_mem, abyss.ANDPeerSessionInfo{
			PeerHash:  info.s.Peer.IDHash(),
			SessionID: info.s.PeerSessionID,
		})
		info.sjnc++
	}

	member_count := 0
	for _, info := range w.peers {
		if info.state != WS_MEM {
			continue
		}
		member_count++
		if len(sjn_mem) != 0 {
			info.s.Peer.TrySendSJN(w.lsid, info.s.PeerSessionID, sjn_mem)
		}
	}

	w.ech <- abyss.NeighborEvent{
		Type:  abyss.ANDTimerRequest,
		Value: rand.Intn(300 * (member_count + 1)),
	}
}

func (w *ANDWorld) RemovePeer(peer abyss.IANDPeer) {
	info, ok := w.peers[peer.IDHash()]
	if !ok {
		return
	}
	switch info.state {
	case WS_CC:
	case WS_JT:
	case WS_JN:
	case WS_RMEM_NJNI:
	case WS_JNI:
	case WS_RMEM:
	case WS_TMEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionClose,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
	case WS_MEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
	}
	delete(w.peers, peer.IDHash())
}
func (w *ANDWorld) Close() {
	for _, info := range w.peers {
		if info.s.Peer != nil {
			info.s.Peer.TrySendRST(w.lsid, uuid.Nil)
		}
		switch info.state {
		case WS_CC:
		case WS_JT:
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDJoinFail,
				LocalSessionID: w.lsid,
				Text:           JNM_CANCELED,
				Value:          JNC_CANCELED,
			}
		case WS_JN:
		case WS_RMEM_NJNI:
		case WS_JNI:
		case WS_RMEM:
		case WS_TMEM:
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionReady,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.s,
			}
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionClose,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.s,
			}
		case WS_MEM:
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionReady,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.s,
			}
		}
	}
	if !w.is_closed {
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDWorldLeave,
			LocalSessionID: w.lsid,
		}
	}
	w.is_closed = true
}
