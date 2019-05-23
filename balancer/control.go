package balancer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pwpon500/caplance/balancer/backend"
	"github.com/vishvananda/netlink"
)

// Balancer is the main data struct for the load balancer
type Balancer struct {
	backends  *backend.BackendHandler // maglev hashtable of backends to their GRE device names
	vip       net.IP                  // VIP the balancer is operating off of
	connectIP net.IP                  // IP for the RPC between backends and balancer
	packets   chan rawPacket          // channel of queued up packets
	stopChan  chan os.Signal          // channel to listen for graceful stop
	listener  *net.IPConn             // vip listener
	testFlag  bool                    // flag to check if we're in test mode
	mux       sync.Mutex              // lock to ensure we don't start and stop at the same time
	ipHeaders [][]byte                // prebuilt ip headers to append onto encapsulated packets
}

type rawPacket struct {
	payload []byte
	size    int
	//	addr    *net.IPAddr
}

// New creates new Balancer. Throws error if capacity is not prime
func New(startVIP, toConnect net.IP, capacity int64) (*Balancer, error) {
	back, err := backend.New(capacity)
	if err != nil {
		return nil, err
	}

	return &Balancer{
		backends:  back,
		vip:       startVIP,
		connectIP: toConnect,
		packets:   make(chan rawPacket),
		stopChan:  make(chan os.Signal),
		testFlag:  false}, nil
}

// NewTest creates new Balancer with the testing flag on
func NewTest(startVIP, toConnect net.IP, capacity int64) (*Balancer, error) {
	back, err := New(startVIP, toConnect, capacity)
	if err != nil {
		return nil, err
	}

	back.testFlag = true

	return back, nil
}

// Add adds a new backend and creates a tunnel between said backend and the LB
func (b *Balancer) Add(name string, ip net.IP) error {
	return b.backends.Add(name, "well hello there")
}

// Start attaches the VIP and starts the load balancer
func (b *Balancer) Start() error {
	b.mux.Lock()
	dev, err := attachVIP(b.vip)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(dev)
	if err != nil {
		return err
	}
	signal.Notify(b.stopChan, syscall.SIGTERM)
	signal.Notify(b.stopChan, syscall.SIGINT)
	go func() {
		graceful := false
		defer func() {
			b.mux.Lock()
			if b.listener != nil {
				b.listener.Close()
			}
			vipNet := &net.IPNet{IP: b.vip, Mask: net.CIDRMask(32, 32)}
			netlink.AddrDel(link, &netlink.Addr{IPNet: vipNet})
			if graceful && !b.testFlag {
				os.Exit(0)
			}
			b.mux.Unlock()
		}()
		sig := <-b.stopChan
		graceful = true
		fmt.Printf("caught sig: %+v \n", sig)
	}()
	b.mux.Unlock()
	b.listen(link)
	return nil
}

// Stop stops the currently running lb by appending onto `stop`
func (b *Balancer) Stop() error {
	if b.listener == nil {
		return errors.New("unstarted balancer cannot be stopped")
	}

	b.stopChan <- syscall.SIGTERM

	return nil
}

// WaitForUnlock waits until the mutex lock is freed
func (b *Balancer) WaitForUnlock() {
	b.mux.Lock()
	b.mux.Unlock()
}
