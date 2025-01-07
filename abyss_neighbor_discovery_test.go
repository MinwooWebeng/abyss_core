package abyss_neighbor_discovery_test

import (
	and "abyss_neighbor_discovery"
	"abyss_neighbor_discovery/tools/dacp"
	"abyss_neighbor_discovery/tools/sear"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

type VirtualHost struct {
	peer_pool  map[string]*VirtualPeer
	id_hash    string
	and        *and.AND
	actionPool dacp.DiscreteActionPool
}

func NewVirtualHost(id_hash string) *VirtualHost {
	result := new(VirtualHost)
	result.peer_pool = make(map[string]*VirtualPeer)
	result.id_hash = id_hash
	result.and = and.NewAND(id_hash)
	result.actionPool = dacp.MakeDiscreteActionPool()
	return result
}

type VirtualPeer struct {
	target_host       *VirtualHost
	owner             *VirtualHost
	is_outbound_alive bool
}

func (p *VirtualPeer) IDHash() string {
	return p.target_host.id_hash
}
func (p *VirtualPeer) TrySendJN(local_session_id uuid.UUID, path string) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendJOK(local_session_id uuid.UUID, peer_session_id uuid.UUID, world_url string, member_sessions []and.PeerSession) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendJDN(peer_session_id uuid.UUID, code int, message string) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendJNI(peer_session_id uuid.UUID, member_session and.PeerSession) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendMEM(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendSNB(peer_session_id uuid.UUID, member_sessions []and.PeerSessionInfo) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendCRR(peer_session_id uuid.UUID, member_sessions []and.PeerSessionInfo) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendRST(local_session_id uuid.UUID, peer_session_id uuid.UUID) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}

type ANDTestGroup struct {
	host_count int
	call_count int

	hosts []*VirtualHost

	remaining_calls      int
	pending_execution    func(option int) //can be nil, if there is no pending action -> then call from actionPool.
	pending_exec_options int              //number of options for pending_execution(option) call. 0 if pending_execution is nil.
	actionPool           dacp.DiscreteActionPool

	run_count int
}

func NewANDTestGroup(host_count int, call_count int) *ANDTestGroup {
	return &ANDTestGroup{
		host_count: host_count,
		call_count: call_count,

		run_count: -1,
	}
}

func (a *ANDTestGroup) Initialize() {
	fmt.Println("------")
	a.run_count++
	a.hosts = make([]*VirtualHost, a.host_count)
	for i := 0; i < a.host_count; i++ {
		host := NewVirtualHost("h" + strconv.Itoa(i))
		a.hosts[i] = host

		var call_action *dacp.DiscreteAction
		call_action = dacp.NewDiscreteAction(func() {
			fmt.Println(host.id_hash + ":callA")
			host.actionPool.AddAction(call_action)
		}, 0)
		host.actionPool.AddAction(call_action)

		var call_action2 *dacp.DiscreteAction
		call_action2 = dacp.NewDiscreteAction(func() {
			fmt.Println(host.id_hash + ":callB")
			host.actionPool.AddAction(call_action2)
		}, 0)
		host.actionPool.AddAction(call_action2)
	}
	a.remaining_calls = a.call_count

	a.actionPool = dacp.MakeDiscreteActionPool()
	var calling_action *dacp.DiscreteAction
	calling_action = dacp.NewDiscreteAction(func() {
		a.pending_exec_options = a.host_count
		a.pending_execution = func(target_host int) {
			target := a.hosts[target_host]

			a.pending_exec_options = target.actionPool.GetActionN()
			a.pending_execution = func(option int) {
				a.pending_exec_options = 0
				a.pending_execution = nil

				action := target.actionPool.PopAction(option)
				action.Exec()
			}
		}

		a.remaining_calls -= 1
		if a.remaining_calls > 0 {
			a.actionPool.AddAction(calling_action)
		}
	}, 0)
	a.actionPool.AddAction(calling_action)
}

func (a *ANDTestGroup) GetInitPaths() int {
	return a.actionPool.GetActionN() // we never start from a pending action. Always call from action pool at first.
}

func (a *ANDTestGroup) Forward(path int) int {
	if a.pending_execution != nil {
		a.pending_execution(path)
	} else {
		action := a.actionPool.PopAction(path)
		action.Exec()
	}

	if a.pending_execution != nil {
		return a.pending_exec_options
	}
	return a.actionPool.GetActionN()
}

func TestAND(t *testing.T) {
	test_group := NewANDTestGroup(3, 3)
	ss := sear.MakeScenarioSearcher(test_group)

	ss.Run()
	fmt.Print("total scenarios: ")
	fmt.Println(test_group.run_count)
}
