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
	WS_RMEM                     //MEM received
	WS_TMEM                     //MEM sent
	WS_MEM                      //member
)

// timestamp is used only for JNI.
type ANDPeerSessionState struct {
	//latest
	abyss.ANDPeerSessionWithTimeStamp
	state int
	sjnp  bool //is sjn suppressed
	sjnc  int  //sjn receive count
}

func NewANDPeerSessionState(peer abyss.IANDPeer, session_id uuid.UUID, timestamp time.Time, state int) *ANDPeerSessionState {
	return &ANDPeerSessionState{
		abyss.ANDPeerSessionWithTimeStamp{
			ANDPeerSession: abyss.ANDPeerSession{
				Peer:          peer,
				PeerSessionID: session_id,
			},
			TimeStamp: timestamp,
		},
		state,
		false,
		0,
	}
}

func (s *ANDPeerSessionState) Clear() {
	s.PeerSessionID = uuid.Nil
	s.TimeStamp = time.Time{}
	if s.Peer != nil {
		s.state = WS_CC
	} else {
		panic("this peer must be removed, not cleared")
	}
	s.sjnp = false
	s.sjnc = 0
}

type ANDWorld struct {
	o *AND //origin (debug purpose)

	local     string //local hash
	lsid      uuid.UUID
	timestamp time.Time
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
		case WS_DC_JNI:
		case WS_CC:
		case WS_JT:
		case WS_JN:
		case WS_RMEM_NJNI:
		case WS_JNI:
		case WS_RMEM:
		case WS_TMEM:
		case WS_MEM:
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
		timestamp: time.Now(),
		join_hash: "",
		join_path: "",
		wurl:      world_url,
		peers:     make(map[string]*ANDPeerSessionState),
		ech:       event_ch,
		is_closed: false,
	}
	for peer_id, peer := range connected_members {
		origin.stat.W(0)

		result.peers[peer_id] = NewANDPeerSessionState(peer, uuid.Nil, time.Time{}, WS_CC)
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
		timestamp: time.Now(),
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
			result.peers[peer_id] = NewANDPeerSessionState(peer, uuid.Nil, time.Time{}, WS_JT)
			peer.TrySendJN(local_session_id, target.Path, result.timestamp)
			continue
		}

		result.peers[peer_id] = NewANDPeerSessionState(peer, uuid.Nil, time.Time{}, WS_CC)
	}
	if !is_join_target_connected {
		result.peers[target.Hash] = NewANDPeerSessionState(nil, uuid.Nil, time.Time{}, WS_DC_JT)
		result.ech <- abyss.NeighborEvent{
			Type:   abyss.ANDConnectRequest,
			Object: target,
		}
	}
	return result
}

func (w *ANDWorld) ClearStates(peer_id string, info *ANDPeerSessionState) {
	switch info.state {
	case WS_DC_JT, WS_DC_JNI:
		delete(w.peers, peer_id)
	case WS_CC:
		info.Clear()
	case WS_JT:
		info.Peer.TrySendRST(w.lsid, info.PeerSessionID)
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDJoinFail,
			LocalSessionID: w.lsid,
			Text:           JNM_INVALID_STATES,
			Value:          JNC_INVALID_STATES,
		}
		info.Clear()
	case WS_JN:
		info.Peer.TrySendJDN(info.PeerSessionID, JNC_INVALID_STATES, JNM_INVALID_STATES)
		info.Clear()
	case WS_MEM:
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionClose,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.ANDPeerSession,
		}
		fallthrough
	case WS_RMEM_NJNI, WS_JNI, WS_RMEM, WS_TMEM:
		info.Peer.TrySendRST(w.lsid, info.PeerSessionID)
		info.Clear()
	}
}

// return (old session ID, success). old session ID is nil if not updated
func (w *ANDWorld) TryUpdateSessionID(s *ANDPeerSessionState, session_id uuid.UUID, timestamp time.Time) bool {
	if s.TimeStamp.Before(timestamp) {
		w.ClearStates(s.Peer.IDHash(), s)
		s.PeerSessionID = session_id
		s.TimeStamp = timestamp
		return true
	} else {
		return false
	}
}
func (w *ANDWorld) IsProperMemberOrReset(info *ANDPeerSessionState, peer_session abyss.ANDPeerSession) bool {
	switch info.state {
	case WS_DC_JT, WS_DC_JNI:
		panic("not connected")
	case WS_MEM:
		if info.PeerSessionID == peer_session.PeerSessionID {
			return true
		}
		fallthrough
	default:
		w.ClearStates(info.Peer.IDHash(), info)
	}
	peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
	return false
}

func (w *ANDWorld) PeerConnected(peer abyss.IANDPeer) {
	info, ok := w.peers[peer.IDHash()]
	if ok { // known peer
		w.o.stat.W(0)

		switch info.state {
		case WS_DC_JT:
			w.o.stat.W(1)

			info.Peer = peer
			peer.TrySendJN(w.lsid, w.join_path, w.timestamp)
			info.state = WS_JT
		case WS_DC_JNI:
			w.o.stat.W(2)

			info.Peer = peer
			info.state = WS_JNI

			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
		default:
			panic("and: duplicate connection")
		}

		return
	}
	//unknown peer
	w.peers[peer.IDHash()] = NewANDPeerSessionState(peer, uuid.Nil, time.Time{}, WS_CC)
}
func (w *ANDWorld) JN(peer_session abyss.ANDPeerSession, timestamp time.Time) {
	info := w.peers[peer_session.Peer.IDHash()]
	switch info.state {
	case WS_CC:
		w.o.stat.W(3)

		info.ANDPeerSession = peer_session
		info.TimeStamp = timestamp
		info.state = WS_JN
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
		}
	case WS_JT: //should not happen. during joining, the world must be hidden, not accepting JN.
		w.o.stat.W(4)

		peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_INVALID_STATES, JNM_INVALID_STATES)
	case WS_JN, WS_RMEM_NJNI, WS_JNI, WS_RMEM, WS_TMEM, WS_MEM:
		w.o.stat.W(5)

		if w.TryUpdateSessionID(info, peer_session.PeerSessionID, timestamp) {
			info.state = WS_JN
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: w.lsid,
				ANDPeerSession: peer_session,
			}
		} else {
			peer_session.Peer.TrySendJDN(peer_session.PeerSessionID, JNC_DUPLICATE, JNM_DUPLICATE) //must not happen
		}
	default:
		panic("and invalid state: JN")
	}
}
func (w *ANDWorld) JOK(peer_session abyss.ANDPeerSession, timestamp time.Time, world_url string, member_infos []abyss.ANDFullPeerSessionIdentity) {
	sender_id := peer_session.Peer.IDHash()
	info := w.peers[sender_id]
	if w.join_hash != sender_id ||
		info.state != WS_JT {
		w.o.stat.W(6)

		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		return
	}

	w.o.stat.W(7)

	info.ANDPeerSession = peer_session
	info.TimeStamp = timestamp
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
		w.JNI_MEMS(sender_id, mem_info)
	}
}
func (w *ANDWorld) JDN(peer abyss.IANDPeer, code int, message string) { //no branch number here... :(
	info := w.peers[peer.IDHash()]
	if w.join_hash != peer.IDHash() ||
		info.state != WS_JT {
		w.o.stat.W(8)

		return
	}

	w.o.stat.W(9)

	w.ech <- abyss.NeighborEvent{
		Type:           abyss.ANDJoinFail,
		LocalSessionID: w.lsid,
		Text:           message,
		Value:          code,
	}
	info.Clear()
}

func (w *ANDWorld) JNI(peer_session abyss.ANDPeerSession, member_info abyss.ANDFullPeerSessionIdentity) {
	sender_id := peer_session.Peer.IDHash()
	info := w.peers[sender_id]

	if !w.IsProperMemberOrReset(info, peer_session) {
		return
	}

	w.JNI_MEMS(sender_id, member_info)
}
func (w *ANDWorld) JNI_MEMS(sender_id string, mem_info abyss.ANDFullPeerSessionIdentity) {
	peer_id := mem_info.AURL.Hash
	if peer_id == w.local {
		w.o.stat.W(10)
		return
	}

	info, ok := w.peers[peer_id]
	if !ok {
		w.o.stat.W(11)

		w.peers[peer_id] = NewANDPeerSessionState(nil, mem_info.SessionID, mem_info.TimeStamp, WS_DC_JNI)
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
	case WS_DC_JT, WS_JT:
		panic("and: proper member check failed (JNI)")
	case WS_DC_JNI:
		w.o.stat.W(12)

		if info.TimeStamp.Before(mem_info.TimeStamp) {
			info.PeerSessionID = mem_info.SessionID
			info.TimeStamp = mem_info.TimeStamp
			info.state = WS_DC_JNI
		}
		//previously, tried connecting. may need to refresh connection trials
	case WS_CC:
		w.o.stat.W(13)

		info.PeerSessionID = mem_info.SessionID
		info.TimeStamp = mem_info.TimeStamp
		info.state = WS_JNI
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionRequest,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.ANDPeerSession,
		}
	case WS_JN:
		w.o.stat.W(14)

		if w.TryUpdateSessionID(info, mem_info.SessionID, mem_info.TimeStamp) {
			//unlikely to happen
			info.state = WS_JNI
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
		}
	case WS_RMEM_NJNI:
		w.o.stat.W(15)

		if w.TryUpdateSessionID(info, mem_info.SessionID, mem_info.TimeStamp) {
			info.state = WS_JNI
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
			return
		}
		if info.PeerSessionID == mem_info.SessionID {
			info.state = WS_RMEM
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
		}
		//else: old session
	case WS_JNI, WS_RMEM, WS_TMEM, WS_MEM:
		if w.TryUpdateSessionID(info, mem_info.SessionID, mem_info.TimeStamp) {
			info.state = WS_JNI
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionRequest,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
			return
		}
	default:
		panic("and invalid state: JNI_MEMS")
	}
}
func (w *ANDWorld) MEM(peer_session abyss.ANDPeerSession, timestamp time.Time) {
	info := w.peers[peer_session.Peer.IDHash()]
	switch info.state {
	case WS_CC:
		w.o.stat.W(16)

		info.ANDPeerSession = peer_session
		info.TimeStamp = timestamp
		info.state = WS_RMEM_NJNI
	case WS_JT:
		w.ClearStates(peer_session.Peer.IDHash(), info)
	case WS_JN, WS_RMEM_NJNI, WS_RMEM, WS_MEM:
		if w.TryUpdateSessionID(info, peer_session.PeerSessionID, timestamp) {
			info.state = WS_RMEM_NJNI
			return
		}
	case WS_JNI:
		if w.TryUpdateSessionID(info, peer_session.PeerSessionID, timestamp) {
			info.state = WS_RMEM_NJNI
			return
		}
		if info.PeerSessionID == peer_session.PeerSessionID {
			info.state = WS_RMEM
		}
	case WS_TMEM:
		w.o.stat.W(17)

		if w.TryUpdateSessionID(info, peer_session.PeerSessionID, timestamp) {
			info.state = WS_RMEM_NJNI
			return
		}
		if info.PeerSessionID == peer_session.PeerSessionID {
			info.state = WS_MEM
			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionReady,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
		}
	default:
		panic("and: impossible disconnected state")
	}
}
func (w *ANDWorld) SJN(peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionIdentity) {
	info := w.peers[peer_session.Peer.IDHash()]
	if !w.IsProperMemberOrReset(info, peer_session) {
		return
	}
	for _, mem_info := range member_infos {
		w.SJN_MEMS(peer_session, mem_info)
	}
}
func (w *ANDWorld) SJN_MEMS(origin abyss.ANDPeerSession, mem_info abyss.ANDPeerSessionIdentity) {
	if mem_info.PeerHash == w.local {
		w.o.stat.W(18)
		return
	}

	info, ok := w.peers[mem_info.PeerHash]
	if ok && info.state == WS_MEM && info.PeerSessionID == mem_info.SessionID {
		info.sjnc++
		return
	}
	origin.Peer.TrySendCRR(w.lsid, origin.PeerSessionID, []abyss.ANDPeerSessionIdentity{mem_info})
}
func (w *ANDWorld) CRR(peer_session abyss.ANDPeerSession, member_infos []abyss.ANDPeerSessionIdentity) {
	info := w.peers[peer_session.Peer.IDHash()]
	if !w.IsProperMemberOrReset(info, peer_session) {
		return
	}
	for _, mem_info := range member_infos {
		w.CRR_MEMS(info, mem_info)
	}
}
func (w *ANDWorld) CRR_MEMS(origin *ANDPeerSessionState, mem_info abyss.ANDPeerSessionIdentity) {
	if mem_info.PeerHash == w.local {
		w.o.stat.W(19)
		return
	}

	info, ok := w.peers[mem_info.PeerHash]
	if ok && info.PeerSessionID == mem_info.SessionID {
		origin.Peer.TrySendJNI(w.lsid, origin.PeerSessionID, info.ANDPeerSessionWithTimeStamp)
		info.Peer.TrySendJNI(w.lsid, info.PeerSessionID, origin.ANDPeerSessionWithTimeStamp)
	}
}
func (w *ANDWorld) SOA(peer_session abyss.ANDPeerSession, objects []abyss.ObjectInfo) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(20)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(21)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDObjectAppend,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
			Object:         objects,
		}
	default:
		w.o.stat.W(22)
	}
}
func (w *ANDWorld) SOD(peer_session abyss.ANDPeerSession, objectIDs []uuid.UUID) {
	info := w.peers[peer_session.Peer.IDHash()]
	if info.PeerSessionID != peer_session.PeerSessionID {
		peer_session.Peer.TrySendRST(w.lsid, peer_session.PeerSessionID)
		w.o.stat.W(23)
		return
	}
	switch info.state {
	case WS_MEM:
		w.o.stat.W(24)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDObjectDelete,
			LocalSessionID: w.lsid,
			ANDPeerSession: peer_session,
			Object:         objectIDs,
		}
	default:
		w.o.stat.W(25)
	}
}
func (w *ANDWorld) RST(peer_session abyss.ANDPeerSession) {
	info := w.peers[peer_session.Peer.IDHash()]
	w.ClearStates(info.Peer.IDHash(), info)
}

func (w *ANDWorld) AcceptSession(peer_session abyss.ANDPeerSession) {
	info, ok := w.peers[peer_session.Peer.IDHash()]
	if !ok {
		w.o.stat.W(26)
		return
	}
	switch info.state {
	case WS_DC_JT:
		panic("and invalid state: AcceptSession")
	case WS_DC_JNI:
	case WS_CC:
		//ignore
	case WS_JT:
		panic("and invalid state: AcceptSession")
	case WS_JN:
		w.o.stat.W(27)

		if info.PeerSessionID != peer_session.PeerSessionID {
			return
		}

		member_infos := make([]abyss.ANDPeerSessionWithTimeStamp, 0)
		for _, p := range w.peers {
			if p.state != WS_MEM {
				continue
			}
			member_infos = append(member_infos, abyss.ANDPeerSessionWithTimeStamp{
				ANDPeerSession: p.ANDPeerSession,
				TimeStamp:      p.TimeStamp,
			})
			p.Peer.TrySendJNI(w.lsid, p.PeerSessionID, info.ANDPeerSessionWithTimeStamp)
		}
		info.Peer.TrySendJOK(w.lsid, info.PeerSessionID, w.timestamp, w.wurl, member_infos)
		info.state = WS_TMEM
	case WS_RMEM_NJNI:
		//ignore
	case WS_JNI:
		w.o.stat.W(28)

		if info.PeerSessionID != peer_session.PeerSessionID {
			return
		}

		info.Peer.TrySendMEM(w.lsid, info.PeerSessionID, w.timestamp)
		info.state = WS_TMEM
	case WS_RMEM:
		w.o.stat.W(29)

		if info.PeerSessionID != peer_session.PeerSessionID {
			return
		}

		info.Peer.TrySendMEM(w.lsid, info.PeerSessionID, w.timestamp)
		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDSessionReady,
			LocalSessionID: w.lsid,
			ANDPeerSession: info.ANDPeerSession,
		}
		info.state = WS_MEM
	case WS_TMEM:
		//ignore
	case WS_MEM:
		//ignore
	default:
		w.o.stat.W(30)
	}
}
func (w *ANDWorld) DeclineSession(peer_session abyss.ANDPeerSession, code int, message string) {
	info, ok := w.peers[peer_session.Peer.IDHash()]
	if !ok {
		w.o.stat.W(31)
		return
	}
	if info.PeerSessionID == peer_session.PeerSessionID {
		//TODO: proper JDN
		w.ClearStates(peer_session.Peer.IDHash(), info)
	}
}
func (w *ANDWorld) TimerExpire() {
	sjn_mem := make([]abyss.ANDPeerSessionIdentity, 0)
	for _, info := range w.peers {
		if info.state != WS_MEM ||
			time.Since(info.TimeStamp) < time.Second ||
			info.sjnp || info.sjnc > 3 {
			continue
		}
		w.o.stat.W(32)

		sjn_mem = append(sjn_mem, abyss.ANDPeerSessionIdentity{
			PeerHash:  info.Peer.IDHash(),
			SessionID: info.PeerSessionID,
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
			w.o.stat.W(33)

			info.Peer.TrySendSJN(w.lsid, info.PeerSessionID, sjn_mem)
		}
	}

	if !w.is_closed {
		w.o.stat.W(34)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDTimerRequest,
			LocalSessionID: w.lsid,
			Value:          300 + rand.Intn(300*(member_count+1)),
		}
	}
}

func (w *ANDWorld) RemovePeer(peer abyss.IANDPeer) {
	w.ClearStates(peer.IDHash(), w.peers[peer.IDHash()])
	delete(w.peers, peer.IDHash())
}
func (w *ANDWorld) Close() {
	for _, info := range w.peers {
		if info.Peer != nil {
			w.o.stat.W(35)
			info.Peer.TrySendRST(w.lsid, uuid.Nil)
		}
		switch info.state {
		case WS_JT:
			w.o.stat.W(36)

			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDJoinFail,
				LocalSessionID: w.lsid,
				Text:           JNM_CANCELED,
				Value:          JNC_CANCELED,
			}
		case WS_MEM:
			w.o.stat.W(37)

			w.ech <- abyss.NeighborEvent{
				Type:           abyss.ANDSessionClose,
				LocalSessionID: w.lsid,
				ANDPeerSession: info.ANDPeerSession,
			}
		default:
			w.o.stat.W(38)
		}
	}
	if !w.is_closed {
		w.o.stat.W(39)

		w.ech <- abyss.NeighborEvent{
			Type:           abyss.ANDWorldLeave,
			LocalSessionID: w.lsid,
		}
	}
	w.is_closed = true
}
