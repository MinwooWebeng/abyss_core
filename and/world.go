package and

import (
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
	WS_RMEM                     //MEM received //**WS_TMEM removed.
	WS_MEM                      //member
)

type ANDPeerSessionState struct {
	p     abyss.IANDPeer
	wsid  uuid.UUID
	state int
	t_mem time.Time //member confirm timestamp
	sjnp  bool      //is sjn suppressed
	sjnc  int       //sjn receive count
}

type ANDWorld struct {
	o *AND //origin (debug purpose)

	local     string //local hash
	lsid      uuid.UUID
	join_hash string                          //const
	join_path string                          //const
	wurl      string                          //const
	peers     map[string]*ANDPeerSessionState //key: hash

	ech       chan abyss.NeighborEvent
	is_closed bool
}

func (w *ANDWorld) CheckSanity() {
	jc := 0
	for peer_id, peer := range w.peers {
		if peer_id == w.local {
			panic("and sanity check failed: loopback connection")
		}

		switch peer.state {
		case WS_DC_JT:
			if peer.p.IDHash() != w.join_hash {
				panic("and sanity check failed: join target mismatch(1)")
			}
			jc++
		case WS_DC_JNI:
			if peer.jsm.Len() == 0 {
				panic("and sanity check failed: jni missing(1)")
			}
		case WS_CC:
		case WS_JT:
			if peer.p.IDHash() != w.join_hash {
				panic("and sanity check failed: join target mismatch(2)")
			}
			jc++
		case WS_JN:
			if peer.jsm.Len() != 0 {
				panic("and sanity check failed: jni not cleared(1)")
			}
		case WS_RMEM_NJNI:
			if peer.jsm.Len() != 0 {
				panic("and sanity check failed: jni-state mismatch")
			}
		case WS_JNI:
			if peer.jsm.Len() == 0 {
				panic("and sanity check failed: jni missing(2)")
			}
		case WS_RMEM:
			if peer.jsm.Len() != 0 {
				panic("and sanity check failed: jni not cleared(2)")
			}
		case WS_TMEM:
			if peer.jsm.Len() == 0 {
				panic("and sanity check failed: jni missing(3)")
			}
		case WS_MEM:
			if peer.jsm.Len() != 0 {
				panic("and sanity check failed: jni not cleared(3)")
			}
		default:
			panic("and sanity check failed: non-existing state")
		}
	}
	if w.join_hash == "" && jc != 0 {
		panic("and sanity check failed: both join and open")
	}
	if jc > 1 {
		panic("and sanity check failed: multiple join targets")
	}
}

func NewWorldOpen(origin *AND, local_hash string, local_session_id uuid.UUID, world_url string, connected_members map[string]abyss.IANDPeer, event_ch chan abyss.NeighborEvent) *ANDWorld {
	result := &ANDWorld{
		o:         origin,
		local:     local_hash,
		lsid:      local_session_id,
		join_hash: "",
		join_path: "",
		wurl:      world_url,
		peers:     make(map[string]*ANDPeerSessionState),
		ech:       event_ch,
		is_closed: false,
	}
	for peer_id, peer := range connected_members {
		origin.stat.W(0)

		result.peers[peer_id] = &ANDPeerSessionState{
			p:     peer,
			jsm:   MakeJNISenderMap(),
			state: WS_CC,
		}
	}
	result.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinSuccess,
		LocalSessionID: local_session_id,
		Text:           world_url,
	}
	result.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDTimerRequest,
		LocalSessionID: result.lsid,
		Value:          500,
	}
	return result
}

func NewWorldJoin(origin *AND, local_hash string, local_session_id uuid.UUID, target *aurl.AURL, connected_members map[string]abyss.IANDPeer, event_ch chan abyss.NeighborEvent) *ANDWorld {
	result := &ANDWorld{
		o:         origin,
		local:     local_hash,
		lsid:      local_session_id,
		join_hash: target.Hash,
		join_path: target.Path,
		peers:     make(map[string]*ANDPeerSessionState),
		ech:       event_ch,
		is_closed: false,
	}
	is_join_target_connected := false
	for peer_id, peer := range connected_members {
		origin.stat.W(1)

		if peer_id == target.Hash {
			is_join_target_connected = true
			result.peers[peer_id] = &ANDPeerSessionState{
				p:     peer,
				jsm:   MakeJNISenderMap(),
				state: WS_JT,
			}
			peer.TrySendJN(local_session_id, target.Path)
			continue
		}

		result.peers[peer_id] = &ANDPeerSessionState{
			p:     peer,
			jsm:   MakeJNISenderMap(),
			state: WS_CC,
		}
	}
	if !is_join_target_connected {
		result.peers[target.Hash] = &ANDPeerSessionState{
			p:     nil,
			jsm:   MakeJNISenderMap(),
			state: WS_DC_JT,
		}
		result.ech <- abyss.NeighborEvent{
			Type:   abyss.ANDConnectRequest,
			Object: target,
		}
	}
	return result
}

func (w *ANDWorld) ClearPeerState(peer_id string) {
	info := w.peers[peer_id]

	if info.p == nil {
		delete(w.peers, peer_id)
		return
	}

	info.jsm.inner.Init()
	info.jsm.direct_wsid = uuid.Nil
	info.state = WS_CC
}

func (w *ANDWorld) PeerConnected(peer abyss.IANDPeer) {
	info, ok := w.peers[peer.IDHash()]
	if ok { // known peer
		w.o.stat.W(0)

		switch info.state {
		case WS_DC_JT:
			w.o.stat.W(1)

			info.p = peer
			peer.TrySendJN(w.lsid, w.join_path)
			info.state = WS_JT
		case WS_DC_JNI:
			w.o.stat.W(2)

			info.p = peer
			info.state = WS_JNI

			for e := info.jsm.inner.Front(); e != nil; e = e.Next() {
				w.ech <- abyss.NeighborEvent{
					Type:           abyss.ANDSessionRequest,
					LocalSessionID: w.lsid,
					ANDPeerSession: abyss.ANDPeerSession{
						Peer:          info.p,
						PeerSessionID: e.Value.(uuid.UUID),
					},
				}
			}
		default:
			panic("and: duplicate connection")
		}

		return
	}
	//unknown peer
	w.peers[peer.IDHash()] = &ANDPeerSessionState{
		p:     peer,
		jsm:   MakeJNISenderMap(),
		state: WS_CC,
	}
}
func (w *ANDWorld) JN(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	switch info.state {
	case WS_CC, WS_JNI:
		w.o.stat.W(3)

		info.jsm.Finalize(peer_session.PeerSessionID)
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: abyss.ANDPeerSession{
				Peer:          info.p,
				PeerSessionID: info.jsm.direct_wsid,
			},
		}
		info.state = WS_JN
	case WS_JT, WS_RMEM_NJNI, WS_RMEM, WS_TMEM, WS_MEM: //may be a simultaneous join, but clients must avoid this from happening; don't reveal a world before gets accepted.
		w.o.stat.W(4)

		peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_INVALID_STATES, JNM_INVALID_STATES)
	case WS_JN:
		w.o.stat.W(5)

		peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_DUPLICATE, JNM_DUPLICATE)
	default:
		panic("and invalid state: JN")
	}
}
func (w *ANDWorld) JOK(peer_session abyss.ANDPeerSession, world_url string, member_infos []abyss.ANDFullPeerSessionIdentity) {
	if w.join_hash != peer_session.Peer.IDHash() {
		w.o.stat.W(6)

		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}

	info := w.peers[peer_session.Peer.IDHash()]
	if info.state != WS_JT {
		w.o.stat.W(7)

		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}

	w.o.stat.W(8)

	info.jsm.Finalize(peer_session.PeerSessionID)
	w.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinSuccess,
		LocalSessionID: w.lsid,
		Text:           world_url,
	}
	w.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDSessionRequest,
		LocalSessionID: w.lsid,
		ANDPeerSession: peer_session,
	}
	info.state = WS_RMEM
	info.sjnp = true

	for _, mem_info := range member_infos {
		w.JNI_MEMS(peer_session.Peer.IDHash(), mem_info)
	}
}
func (w *ANDWorld) JDN(peer abyss.IANDPeer, code int, message string) { //no branch number here... :(
	peer_id := peer.IDHash()
	if w.join_hash != peer_id {
		w.o.stat.W(9)

		return
	}

	info := w.peers[peer_id]
	if info.state != WS_JT {
		w.o.stat.W(10)

		return
	}

	w.o.stat.W(11)

	w.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinFail,
		LocalSessionID: w.lsid,
		Text:           message,
		Value:          code,
	}
	w.ClearPeerState(peer_id)
}
func (w *ANDWorld) JNI(peer_session abyss.ANDPeerSession, member_info abyss.ANDFullPeerSessionIdentity) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.jsm.direct_wsid != peer_session.PeerSessionID ||
		info.state != WS_MEM {
		w.o.stat.W(12)

		//client fatal error: JNI can only be received from a WS_MEM

		peer_session.Peer.TrySendRST(w.lsid, uuid.Nil)
		w.ClearPeerState(peer_session.Peer.IDHash())
		return
	}

	w.JNI_MEMS(peer_session.Peer.IDHash(), member_info)
}
func (w *ANDWorld) JNI_MEMS(sender_id string, mem_info abyss.ANDFullPeerSessionIdentity) {
	peer_id := mem_info.AURL.Hash
	if peer_id == w.local {
		w.o.stat.W(13)
		return
	}

	info, ok := w.peers[peer_id]
	if !ok {
		w.o.stat.W(14)

		w.peers[peer_id] = &ANDPeerSessionState{
			p:     nil,
			state: WS_DC_JNI,
			sjnp:  false,
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
	case WS_DC_JT:
		w.o.stat.W(15)
		// should not reach here
	case WS_DC_JNI:
		w.o.stat.W(16)

		info.jsm.Update(sender_id, mem_info.SessionID)
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: abyss.ANDPeerSession{
				Peer:          info.p,
				PeerSessionID: mem_info.SessionID,
			},
		}
	case WS_CC:
		w.o.stat.W(17)

		info.jsm.Update(sender_id, mem_info.SessionID)
		info.state = WS_JNI
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: abyss.ANDPeerSession{
				Peer:          info.p,
				PeerSessionID: mem_info.SessionID,
			},
		}
	case WS_JT, WS_JN:
		//early join/rejoin - hardly possible? ignore
	case WS_RMEM_NJNI:
		w.o.stat.W(18)

		if info.jsm.direct_wsid != mem_info.SessionID {
			//expired jni
			return
		}
		info.state = WS_RMEM
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: abyss.ANDPeerSession{
				Peer:          info.p,
				PeerSessionID: mem_info.SessionID,
			},
		}
	case WS_JNI:
		info.jsm.Update(sender_id, mem_info.SessionID)
	case WS_RMEM, WS_MEM:
		//duplicate JNI, session ID already confirmed. ignore
	default:
		panic("and invalid state: JNI_MEMS")
	}
}
func (w *ANDWorld) MEM(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	switch info.state {
	case WS_CC:
		w.o.stat.W(19)

		info.s.PeerSessionID = peer_session.PeerSessionID
		info.state = WS_RMEM_NJNI
	case WS_JNI:
		w.o.stat.W(20)

		if info.s.PeerSessionID != peer_session.PeerSessionID {
			info.s.PeerSessionID = peer_session.PeerSessionID
			info.state = WS_RMEM_NJNI //likely, expired JNI
			return
		}

		info.state = WS_RMEM
	case WS_TMEM:
		w.o.stat.W(21)

		if info.s.PeerSessionID != peer_session.PeerSessionID {
			info.s.PeerSessionID = peer_session.PeerSessionID
			peer_session.Peer.TrySendRST(w.lsid, info.s.PeerSessionID)
			info.state = WS_RMEM_NJNI
			return
		}

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.t_mem = time.Now()
		info.state = WS_MEM
	case WS_RMEM_NJNI, WS_RMEM, WS_MEM:
		w.o.stat.W(22)

		peer_session.Peer.TrySendRST(w.lsid, info.s.PeerSessionID)
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		info.state = WS_CC
	case WS_JT, WS_JN:
		w.o.stat.W(23)

	default:
		panic("and: impossible disconnected state")
	}
}
func (w *ANDWorld) SJN(peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionIdentity) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		w.o.stat.W(24)

		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(25)

		for _, mem_info := range member_infos {
			w.SJN_MEMS(peer_session, mem_info)
		}
	default:
		w.o.stat.W(26)

		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
	}
}
func (w *ANDWorld) SJN_MEMS(origin abyss.ANDPeerSession, mem_info abyss.ANDPeerSessionIdentity) {
	if mem_info.PeerHash == w.local {
		w.o.stat.W(27)
		return
	}

	info, ok := w.peers[mem_info.PeerHash]
	if !ok {
		origin.Peer.TrySendCRR(w.lsid, origin.PeerSessionID, []abyss.ANDPeerSessionIdentity{mem_info})
		w.o.stat.W(28)
		return
	}
	if info.s.PeerSessionID != mem_info.SessionID {
		w.o.stat.W(29)
		return
	}
	switch info.state {
	case WS_JNI, WS_RMEM, WS_TMEM, WS_MEM:
		w.o.stat.W(30)

		info.sjnc++
	default:
		w.o.stat.W(31)

		origin.Peer.TrySendCRR(w.lsid, origin.PeerSessionID, []abyss.ANDPeerSessionIdentity{mem_info})
	}
}
func (w *ANDWorld) CRR(peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionIdentity) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(32)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(33)

		for _, mem_info := range member_infos {
			w.CRR_MEMS(peer_session, mem_info)
		}
	default:
		w.o.stat.W(34)

		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
	}
}
func (w *ANDWorld) CRR_MEMS(origin abyss.ANDPeerSession, mem_info abyss.ANDPeerSessionIdentity) {
	if mem_info.PeerHash == w.local {
		w.o.stat.W(35)
		return
	}

	info := w.peers[mem_info.PeerHash]
	if info.s.PeerSessionID != mem_info.SessionID {
		w.o.stat.W(36)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(37)
		origin.Peer.TrySendJNI(w.lsid, origin.PeerSessionID, info.s)
		info.s.Peer.TrySendJNI(w.lsid, info.s.PeerSessionID, origin)
	default:
		w.o.stat.W(38)
	}
}
func (w *ANDWorld) SOA(peer_session abyss.ANDPeerSession, objects []abyss.ObjectInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(39)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(40)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDObjectAppend,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
			Object:         objects,
		}
	default:
		w.o.stat.W(41)
	}
}
func (w *ANDWorld) SOD(peer_session abyss.ANDPeerSession, objectIDs []uuid.UUID) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(42)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(43)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDObjectDelete,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
			Object:         objectIDs,
		}
	default:
		w.o.stat.W(44)
	}
}
func (w *ANDWorld) RST(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		w.o.stat.W(45)
		return
	}
	switch info.state {
	case WS_JT:
		w.o.stat.W(46)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDJoinFail,
			LocalSessionID: w.lsid,
			Text:           JNM_CANCELED,
			Value:          JNC_CANCELED,
		}
	case WS_TMEM:
		w.o.stat.W(47)

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
		w.o.stat.W(48)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionClose,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
		}
	default:
		w.o.stat.W(49)
	}
	delete(w.peers, peer_session.Peer.IDHash())
}

func (w *ANDWorld) AcceptSession(peer_session abyss.ANDPeerSession) {
	info, ok := w.peers[peer_session.Peer.IDHash()]
	if !ok {
		w.o.stat.W(50)
		return
	}
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(51)
		return
	}
	switch info.state {
	case WS_JN:
		w.o.stat.W(52)

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
	case WS_JNI:
		w.o.stat.W(53)

		info.s.Peer.TrySendMEM(w.lsid, info.s.PeerSessionID)
		info.state = WS_TMEM
	case WS_RMEM:
		w.o.stat.W(54)

		info.s.Peer.TrySendMEM(w.lsid, info.s.PeerSessionID)
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
		info.t_mem = time.Now()
		info.state = WS_MEM
	default:
		w.o.stat.W(55)
	}
}
func (w *ANDWorld) DeclineSession(peer_session abyss.ANDPeerSession, code int, message string) {
	info, ok := w.peers[peer_session.Peer.IDHash()]
	if !ok {
		w.o.stat.W(56)
		return
	}
	if info.s.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(57)
		return
	}
	switch info.state {
	default:
		w.o.stat.W(58)
	}
}
func (w *ANDWorld) TimerExpire() {
	sjn_mem := make([]abyss.ANDPeerSessionIdentity, 0)
	for _, info := range w.peers {
		if info.state != WS_MEM ||
			time.Since(info.t_mem) < time.Second ||
			info.sjnp || info.sjnc > 3 {
			continue
		}
		w.o.stat.W(59)

		sjn_mem = append(sjn_mem, abyss.ANDPeerSessionIdentity{
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
			w.o.stat.W(60)

			info.s.Peer.TrySendSJN(w.lsid, info.s.PeerSessionID, sjn_mem)
		}
	}

	if !w.is_closed {
		w.o.stat.W(61)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDTimerRequest,
			LocalSessionID: w.lsid,
			Value:          min(300, rand.Intn(300*(member_count+1))),
		}
	}
}

func (w *ANDWorld) RemovePeer(peer abyss.IANDPeer) {
	info, ok := w.peers[peer.IDHash()]
	if !ok {
		w.o.stat.W(62)
		return
	}
	switch info.state {
	case WS_TMEM:
		w.o.stat.W(63)

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
		w.o.stat.W(64)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.s,
		}
	default:
		w.o.stat.W(65)
	}
	delete(w.peers, peer.IDHash())
}
func (w *ANDWorld) Close() {
	for _, info := range w.peers {
		if info.s.Peer != nil {
			w.o.stat.W(66)
			info.s.Peer.TrySendRST(w.lsid, uuid.Nil)
		}
		switch info.state {
		case WS_JT:
			w.o.stat.W(67)

			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDJoinFail,
				LocalSessionID: w.lsid,
				Text:           JNM_CANCELED,
				Value:          JNC_CANCELED,
			}
		case WS_TMEM:
			w.o.stat.W(68)

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
			w.o.stat.W(69)

			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionClose,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.s,
			}
		default:
			w.o.stat.W(70)
		}
	}
	if !w.is_closed {
		w.o.stat.W(71)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDWorldLeave,
			LocalSessionID: w.lsid,
		}
	}
	w.is_closed = true
}
