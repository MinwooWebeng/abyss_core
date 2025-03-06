package test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"abyss_neighbor_discovery/aurl"
	abyss_host "abyss_neighbor_discovery/host"
	abyss "abyss_neighbor_discovery/interfaces"
	"abyss_neighbor_discovery/tools/functional"

	"github.com/google/uuid"
)

func printWorldEvents(prefix string, host abyss.IAbyssHost, world abyss.IAbyssWorld, dwell_time_ms int, joiner_ch chan string, fin_ch chan bool) {
	ev_ch := world.GetEventChannel()

	for {
		select {
		case <-time.After(time.Duration(dwell_time_ms) * time.Millisecond):
			host.LeaveWorld(world)
			fmt.Println(prefix + " Left World: " + world.SessionID().String())
			fin_ch <- true
			return
		case joiner_aurl := <-joiner_ch:
			parsed_aurl, err := aurl.ParseAURL(joiner_aurl)
			if err != nil {
				panic(err.Error())
			}
			host.OpenOutboundConnection(parsed_aurl)
		case event_unknown := <-ev_ch:
			switch event := event_unknown.(type) {
			case abyss.EWorldPeerRequest:
				fmt.Println(prefix + " accepting " + event.PeerHash)
				event.Accept()
			case abyss.EWorldPeerReady:
				fmt.Println(prefix + " peer ready: " + event.Peer.Hash())
				//event.Peer.AppendObjects([]abyss.ObjectInfo{abyss.ObjectInfo{ID: uuid.New(), Address: "https://abyssal.com/cat.obj"}})
			case abyss.EPeerObjectAppend:
				fmt.Println(prefix + " " + event.PeerHash + " appended" + functional.Accum_all(event.Objects, "", func(obj abyss.ObjectInfo, accum string) string {
					return accum + " " + obj.ID.String() + "|" + obj.Address
				}))
			case abyss.EPeerObjectDelete:
				fmt.Println(prefix + " " + event.PeerHash + " deleted" + functional.Accum_all(event.ObjectIDs, "", func(obj uuid.UUID, accum string) string { return accum + " " + obj.String() }))
			case abyss.EWorldPeerLeave:
				fmt.Println(prefix + " peer leave: " + event.PeerHash)
			default:
				panic("unknown world event")
			}
		}
	}
}

func TestHost(t *testing.T) {
	hostA, hostA_pathMap := abyss_host.NewBetaAbyssHost("hostA")
	hostB, _ := abyss_host.NewBetaAbyssHost("hostB")

	go hostA.ListenAndServe(context.Background())
	go hostB.ListenAndServe(context.Background())

	A_world, err := hostA.OpenWorld("http://a.world.com")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("[hostA] Opened World: " + A_world.SessionID().String())
	hostA_pathMap.SetMapping("/home", A_world.SessionID()) //this opens the world for join from A's side
	A_world_fin := make(chan bool)
	go printWorldEvents("[hostA]", hostA, A_world, 30000, make(chan string), A_world_fin)

	<-time.After(time.Second)
	hostA.OpenOutboundConnection(hostB.GetLocalAbyssURL())

	join_url := hostA.GetLocalAbyssURL()
	join_url.Path = "/home"

	fmt.Println("[hostB] Joining World")
	join_ctx, join_ctx_cancel := context.WithTimeout(context.Background(), 20*time.Second)
	B_A_world, err := hostB.JoinWorld(join_ctx, join_url)
	join_ctx_cancel()

	if err != nil {
		fmt.Println("[hostB] Join Failed:::" + err.Error())
		return
	}
	fmt.Println("[hostB] Joined World: " + B_A_world.SessionID().String())

	B_A_world_fin := make(chan bool)
	go printWorldEvents("[hostB]", hostB, B_A_world, 30000, make(chan string), B_A_world_fin)

	<-A_world_fin
	<-B_A_world_fin
}

func RandomBoolean() bool {
	return rand.Intn(2) == 1
}

func RandomString() string {
	chars := []rune("abcdefghijklmnopqrstuvwxyz")

	// Create a slice of runes to hold our random characters
	result := make([]rune, 3)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

type GlobalWorldRegistry struct {
	list map[string]struct {
		join_aurl string
		wait_ch   chan string
	} //some key -> aurl.
	mtx *sync.Mutex
}

func NewGlobalWorldRegistry() *GlobalWorldRegistry {
	return &GlobalWorldRegistry{
		list: make(map[string]struct {
			join_aurl string
			wait_ch   chan string
		}),
		mtx: new(sync.Mutex),
	}
}
func (g *GlobalWorldRegistry) Append(key, entry string, waiter_ch chan string) {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if _, ok := g.list[key]; ok {
		panic("same key in GlobalWorldRegistry")
	}
	g.list[key] = struct {
		join_aurl string
		wait_ch   chan string
	}{entry, waiter_ch}
}

func (g *GlobalWorldRegistry) Delete(key string) {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	delete(g.list, key)
}
func (g *GlobalWorldRegistry) RequestRandom(local_aurl string) (string, bool) {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if len(g.list) == 0 {
		return "", false
	}

	keys := make([]string, 0, len(g.list))
	for k := range g.list {
		keys = append(keys, k)
	}

	targ_key := keys[rand.Intn(len(g.list))]
	entry := g.list[targ_key]
	entry.wait_ch <- local_aurl
	return entry.join_aurl, true
}

type AutonomousHost struct {
	global_join_targets *GlobalWorldRegistry
	abyss_host          *abyss_host.AbyssHost
	abyss_pathMap       *abyss_host.DefaultPathResolver
	log_prefix          string
}

func (a *AutonomousHost) Run(ctx context.Context, done_ch chan bool) {
	defer func() {
		done_ch <- true
	}()
	go a.abyss_host.ListenAndServe(context.Background())
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if RandomBoolean() {
				world_name := RandomString()
				world, err := a.abyss_host.OpenWorld("https://" + world_name + ".com")
				if err != nil {
					panic(err.Error())
				}
				fmt.Println(a.log_prefix + " Opened world: " + world.SessionID().String() + "/" + world_name)
				a.abyss_pathMap.SetMapping("/"+world_name, world.SessionID())

				raw_aurl := a.abyss_host.GetLocalAbyssURL()
				raw_aurl.Path = "/" + world_name
				joiner_ch := make(chan string)
				a.global_join_targets.Append(a.log_prefix+world_name, raw_aurl.ToString(), joiner_ch)

				printWorldEvents(a.log_prefix, a.abyss_host, world, rand.Intn(10000), joiner_ch, make(chan bool, 1))

				a.abyss_pathMap.DeleteMapping("/" + world_name)
				a.global_join_targets.Delete(a.log_prefix + world_name)
			} else {
				join_target_string, ok := a.global_join_targets.RequestRandom(a.abyss_host.GetLocalAbyssURL().ToString())
				if !ok {
					continue
				}
				join_target, err := aurl.ParseAURL(join_target_string)
				if err != nil {
					panic(err.Error())
				}

				fmt.Println(a.log_prefix + " Joining world: " + join_target.ToString())
				join_ctx, join_ctx_cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				world, err := a.abyss_host.JoinWorld(join_ctx, join_target)
				if err != nil {
					fmt.Println("join fail: " + err.Error())
					join_ctx_cancel()
					return
				}
				join_ctx_cancel()

				fmt.Println(a.log_prefix + " Joined world: " + world.SessionID().String())
				world_name := RandomString()
				a.abyss_pathMap.SetMapping("/"+world_name, world.SessionID())

				raw_aurl := a.abyss_host.GetLocalAbyssURL()
				raw_aurl.Path = "/" + world_name
				joiner_ch := make(chan string)
				a.global_join_targets.Append(a.log_prefix+world_name, raw_aurl.ToString(), joiner_ch)

				printWorldEvents(a.log_prefix, a.abyss_host, world, rand.Intn(30000), joiner_ch, make(chan bool, 1))

				a.abyss_pathMap.DeleteMapping("/" + world_name)
				a.global_join_targets.Delete(a.log_prefix + world_name)
			}
			fmt.Println(a.log_prefix + "next round")
		}
	}
}

func TestMoreHosts(t *testing.T) {
	N_hosts := 5

	global_world_reg := NewGlobalWorldRegistry()

	ctx, ctx_cancel := context.WithTimeout(context.Background(), 30*time.Second)

	hosts := make([]*AutonomousHost, N_hosts)
	done_ch := make(chan bool, N_hosts)
	for i := range N_hosts {
		host_name := "host" + strconv.Itoa(i+1)

		host, host_pathMap := abyss_host.NewBetaAbyssHost(host_name)

		hosts[i] = &AutonomousHost{
			global_join_targets: global_world_reg,
			abyss_host:          host,
			abyss_pathMap:       host_pathMap,
			log_prefix:          "[" + host_name + "] ",
		}
		go hosts[i].Run(ctx, done_ch)
	}

	for range N_hosts {
		<-done_ch
	}
	ctx_cancel()
}
