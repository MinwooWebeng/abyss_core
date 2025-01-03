package abyss_neighbor_discovery

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

type DiscreteAction struct {
	id           int
	action       func()
	precursor_id int
}

var action_counter int

func NewDiscreteAction(action func(), precursor_id int) *DiscreteAction {
	result := new(DiscreteAction)
	action_counter++
	result.id = action_counter
	result.action = action
	result.precursor_id = precursor_id
	return result
}

type DiscreteActionPool struct {
	actions_ready   []*DiscreteAction
	actions_pending []*DiscreteAction
}

func MakeDiscreteActionPool() DiscreteActionPool {
	return DiscreteActionPool{actions_ready: make([]*DiscreteAction, 0), actions_pending: make([]*DiscreteAction, 0)}
}
func (p *DiscreteActionPool) PopAction(index int) *DiscreteAction {
	result := p.actions_ready[index]
	p.actions_ready = append(p.actions_ready[:index], p.actions_ready[index+1:]...)

	stil_pending_actions := make([]*DiscreteAction, 0)
	for _, pending_action := range p.actions_pending {
		if pending_action.precursor_id == result.id {
			p.actions_ready = append(p.actions_ready, pending_action)
		} else {
			stil_pending_actions = append(stil_pending_actions, pending_action)
		}
	}
	p.actions_pending = stil_pending_actions

	return result
}
func (p *DiscreteActionPool) AddAction(action *DiscreteAction) int {
	for _, ra := range p.actions_ready {
		if ra.id == action.precursor_id {
			p.actions_pending = append(p.actions_pending, action)
			return action.id
		}
	}
	for _, pa := range p.actions_pending {
		if pa.id == action.precursor_id {
			p.actions_pending = append(p.actions_pending, action)
			return action.id
		}
	}

	p.actions_ready = append(p.actions_ready, action)
	return action.id
}

func (p *DiscreteActionPool) GetActionN() int {
	return len(p.actions_ready)
}

func TestActionPool(t *testing.T) {
	expected_array := []int{4, 5, 1, 2, 3}
	result_array := make([]int, 0, 10)

	action_pool := MakeDiscreteActionPool()
	r1 := action_pool.AddAction(NewDiscreteAction(func() { result_array = append(result_array, 1) }, 0))
	r2 := action_pool.AddAction(NewDiscreteAction(func() { result_array = append(result_array, 2) }, r1))
	action_pool.AddAction(NewDiscreteAction(func() { result_array = append(result_array, 3) }, r2))
	r4 := action_pool.AddAction(NewDiscreteAction(func() { result_array = append(result_array, 4) }, 0))
	action_pool.AddAction(NewDiscreteAction(func() { result_array = append(result_array, 5) }, r4))

	for len(action_pool.actions_ready) != 0 {
		action := action_pool.PopAction(len(action_pool.actions_ready) - 1)
		action.action()
	}

	if !reflect.DeepEqual(expected_array, result_array) {
		result_string := ""
		for _, r := range result_array {
			result_string += strconv.Itoa(r) + " "
		}
		t.Fatalf("%s", "result not matching: [ "+result_string+"]")
	}
}

type ScenarioNode struct {
	parent      *ScenarioNode
	children    []*ScenarioNode
	is_final    bool //is children determined
	is_searched bool //is child paths all searched
}

func NewScenarioNode(parent *ScenarioNode) *ScenarioNode {
	result := new(ScenarioNode)
	result.parent = parent
	result.children = make([]*ScenarioNode, 0)
	result.is_final = false
	result.is_searched = false
	return result
}

func (s *ScenarioNode) OpenScenarioPaths(num_paths int) {
	if s.is_final {
		if len(s.children) != num_paths {
			panic("scenario path overwritten")
		}
		return
	}

	for i := 0; i < num_paths; i++ {
		s.children = append(s.children, NewScenarioNode(s))
	}

	s.is_final = true
}

type ScenarioMap struct {
	root         *ScenarioNode
	current_node *ScenarioNode
}

func NewScenarioMap() *ScenarioMap {
	result := new(ScenarioMap)
	result.root = NewScenarioNode(nil)
	result.current_node = result.root
	return result
}

func _mark_searched_parent_nodes(current_node *ScenarioNode) {
	if current_node.parent == nil { // current_node is root node
		return
	}

	if current_node.parent.children[len(current_node.parent.children)-1] == current_node {
		current_node.parent.is_searched = true
		_mark_searched_parent_nodes(current_node.parent)
	}
}

func (s *ScenarioMap) TryGetNextSearchBranch() (int, bool, bool) { // branch, is_end, ok
	if !s.current_node.is_final {
		panic("entered unfinialized path")
	}

	for i, child := range s.current_node.children {
		if child.is_searched {
			continue
		}

		if len(child.children) == 0 && child.is_final { // check if child is a leaf node.
			child.is_searched = true
			_mark_searched_parent_nodes(child)
			s.current_node = s.root
			return i, true, true
		} else {
			s.current_node = child // child may not be finialized.
			return i, false, true
		}
	}

	if len(s.current_node.children) == 0 {
		s.current_node.is_searched = true
		_mark_searched_parent_nodes(s.current_node)
		s.current_node = s.root
		return -1, true, true
	}

	return -1, false, false
}

func StringyfyPath(paths []int) string {
	result := ""
	for _, path := range paths {
		result += strconv.Itoa(path)
		result += " "
	}
	return result
}

func TestScenarioMap(t *testing.T) {
	ss := NewScenarioMap()
	ss.root.OpenScenarioPaths(3)
	ss.root.children[0].OpenScenarioPaths(0)
	ss.root.children[1].OpenScenarioPaths(1)
	ss.root.children[1].children[0].OpenScenarioPaths(0)
	ss.root.children[2].OpenScenarioPaths(2)
	ss.root.children[2].children[0].OpenScenarioPaths(0)

	for i, entry := range []int{0, 1, 0, 2, 0, 2, 1} {
		branch, is_end, ok := ss.TryGetNextSearchBranch()
		if !ok {
			t.Fatalf("incomplete search")
		}
		switch i {
		case 0, 2, 4:
			if !is_end {
				t.Fatalf("%s", "is_end not detected in: "+strconv.Itoa(i))
			}
		default:
			if is_end {
				t.Fatalf("%s", "is_end false positive in: "+strconv.Itoa(i))
			}
		}
		if branch != entry {
			t.Fatalf("%s", "expected: "+strconv.Itoa(entry)+" , got "+strconv.Itoa(branch))
		}
	}

	ss.root.children[2].children[1].OpenScenarioPaths(0)

	branch, is_end, ok := ss.TryGetNextSearchBranch()
	if branch != -1 || !is_end || !ok {
		t.Fatalf("failed to detect newly finialized path")
	}
}

type IDecisionMachine interface {
	Initialize()
	GetInitPaths() int
	Forward(path int) int //return number of next branches
}

type ScenarioSearcher struct {
	scenario_map *ScenarioMap
	machine      IDecisionMachine
}

func MakeScenarioSearcher(machine IDecisionMachine) ScenarioSearcher {
	result := ScenarioSearcher{}
	result.scenario_map = NewScenarioMap()
	result.machine = machine
	return result
}

func (s *ScenarioSearcher) Run() {
	s.machine.Initialize()
	s.scenario_map.root.OpenScenarioPaths(s.machine.GetInitPaths())

	running := true
	for running {
		s.machine.Initialize()
		target_branch := s.scenario_map.root

		for {
			branch, _, ok := s.scenario_map.TryGetNextSearchBranch()
			if !ok {
				running = false
				break
			}

			if branch == -1 {
				break
			}

			target_branch = target_branch.children[branch]
			next_paths := s.machine.Forward(branch)
			target_branch.OpenScenarioPaths(next_paths)
		}
	}
}

type TestDecisionMachine struct {
	paths            map[int]int
	current_location int
	perm_log         []int
}

func (m *TestDecisionMachine) Initialize() {
	m.current_location = -1
}

func (m *TestDecisionMachine) GetInitPaths() int {
	return 3
}

func (m *TestDecisionMachine) Forward(path int) int {
	if m.current_location == -1 {
		m.current_location = path
	} else {
		m.current_location = m.current_location*10 + path
	}

	m.perm_log = append(m.perm_log, m.current_location)

	num_paths := m.paths[m.current_location]
	return num_paths
}

func MakeTestDecisionMachine() *TestDecisionMachine {
	result := new(TestDecisionMachine)
	result.paths = make(map[int]int)
	result.paths[0] = 0
	result.paths[1] = 2
	result.paths[10] = 1
	result.paths[100] = 2
	result.paths[1000] = 0
	result.paths[1001] = 0
	result.paths[11] = 0
	result.paths[2] = 2
	result.paths[20] = 0
	result.paths[21] = 3
	result.paths[210] = 0
	result.paths[211] = 0
	result.paths[212] = 0
	result.perm_log = make([]int, 0, 10)
	return result
}

func TestScenarioSearcher(t *testing.T) {
	machine := MakeTestDecisionMachine()
	searcher := MakeScenarioSearcher(machine)

	searcher.Run()
	if StringyfyPath(machine.perm_log) != "0 1 10 100 1000 1 10 100 1001 1 11 2 20 2 21 210 2 21 211 2 21 212 " {
		t.Fatalf("search failed")
	}
}

type VirtualHost struct {
	peer_pool map[string]*VirtualPeer
	id_hash   string
	and       *AND
}

func NewVirtualHost(id_hash string) *VirtualHost {
	result := new(VirtualHost)
	result.peer_pool = make(map[string]*VirtualPeer)
	result.id_hash = id_hash
	result.and = NewAND(id_hash)
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
func (p *VirtualPeer) TrySendJOK(local_session_id uuid.UUID, peer_session_id uuid.UUID, world_url string, member_sessions []PeerSession) bool {
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
func (p *VirtualPeer) TrySendJNI(peer_session_id uuid.UUID, member_session PeerSession) bool {
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
func (p *VirtualPeer) TrySendSNB(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool {
	if !p.is_outbound_alive {
		return false
	}

	return true
}
func (p *VirtualPeer) TrySendCRR(peer_session_id uuid.UUID, member_sessions []PeerSessionInfo) bool {
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

type ANDTester struct {
	hosts      []*VirtualHost
	actionPool DiscreteActionPool

	scenarioMap *ScenarioMap
}

func MakeANDTester() ANDTester {
	result := ANDTester{}
	result.scenarioMap = NewScenarioMap()
	return result
}

func (a *ANDTester) InitializeHosts(host_count int) {
	a.hosts = make([]*VirtualHost, host_count)
	for i := 0; i < host_count; i++ {
		a.hosts[i] = NewVirtualHost("host_" + strconv.Itoa(i))
	}
	a.actionPool = MakeDiscreteActionPool()
}
