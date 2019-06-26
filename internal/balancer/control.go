package balancer

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AkihiroSuda/go-netfilter-queue"
	"github.com/coreos/go-iptables/iptables"
	"github.com/pwpon500/caplance/internal/balancer/backends"
	"github.com/vishvananda/netlink"
)

// Balancer is the main data struct for the load balancer
type Balancer struct {
	backendManager *backends.Manager  // manager for backends
	vip            net.IP             // VIP the balancer is operating off of
	connectIP      net.IP             // IP for the RPC between backends and balancer
	packets        chan []byte        // channel of queued up packets
	stopChan       chan os.Signal     // channel to listen for graceful stop
	testFlag       bool               // flag to check if we're in test mode
	mux            sync.Mutex         // lock to ensure we don't start and stop at the same time
	nfq            *netfilter.NFQueue // queue to grab packets from the iptables nfqueue
}

// New creates new Balancer. Throws error if capacity is not prime
func New(startVIP, toConnect net.IP, capacity int) (*Balancer, error) {
	manager, err := backends.NewManager(toConnect, 1338, capacity)
	if err != nil {
		return nil, err
	}

	return &Balancer{
		backendManager: manager,
		vip:            startVIP,
		connectIP:      toConnect,
		packets:        make(chan []byte, 100),
		stopChan:       make(chan os.Signal, 5),
		testFlag:       false}, nil
}

// NewTest creates new Balancer with the testing flag on
func NewTest(startVIP, toConnect net.IP, capacity int) (*Balancer, error) {
	back, err := New(startVIP, toConnect, capacity)
	if err != nil {
		return nil, err
	}

	back.testFlag = true

	return back, nil
}

/*
	Add is currently deprecated in favor of a connect-based approach

// Add adds a new backend and creates a tunnel between said backend and the LB
func (b *Balancer) Add(name string, ip net.IP) error {
	return b.backendMap.Add(name, ip)
}
*/

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

			log.Println(1)
			if b.nfq != nil {
				go b.nfq.Close()                   // need to multithread because sometimes b.nfq.Close() blocks indefinitely
				time.Sleep(500 * time.Millisecond) // give nfq some time to close
			}

			log.Println(2)
			allBackends := b.backendManager.GetBackends()
			for _, back := range allBackends {
				back.Writer.Close()
			}

			log.Println(3)
			vipNet := &net.IPNet{IP: b.vip, Mask: net.CIDRMask(32, 32)}
			netlink.AddrDel(link, &netlink.Addr{IPNet: vipNet})

			log.Println(4)
			ipt, err := iptables.New()
			if err != nil {
				log.Println(err)
			}
			ipt.Delete("filter", "INPUT", "-j", "NFQUEUE", "--queue-num", "0", "-d", b.vip.String(), "-p", "tcp")
			ipt.Delete("filter", "INPUT", "-j", "NFQUEUE", "--queue-num", "0", "-d", b.vip.String(), "-p", "udp")

			log.Println(5)
			if graceful && !b.testFlag {
				log.Println("Exiting")
				os.Exit(0)
			}

			b.mux.Unlock()
		}()
		sig := <-b.stopChan
		graceful = true
		log.Printf("caught sig: %+v \n", sig)
		b.stopChan <- sig
	}()

	b.mux.Unlock()
	var wg sync.WaitGroup
	wg.Add(2)
	go b.backendManager.Listen()
	go b.listen()
	wg.Wait()
	return nil
}

// Stop stops the currently running lb by appending onto `stop`
func (b *Balancer) Stop() error {
	if b.nfq == nil {
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
