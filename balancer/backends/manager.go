package backends

import (
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/pwpon500/caplance/util"
)

const (
	REGISTER_TIMEOUT = 10
	READ_TIMEOUT     = 30
	WRITE_TIMEOUT    = 5
)

type managedBackend struct {
	name   string
	dataIP net.IP
	comm   util.Communicator
}

// Manager contains the info needed to manage the backends
type Manager struct {
	listener        net.Listener               // listener for new backends
	listenIP        net.IP                     // ip to listen on
	listenPort      int                        // port to listen on
	handler         *Handler                   // handler for backends
	managedBackends map[string]*managedBackend // map of backend name to its communicator
}

// NewManager instantiates a new instance of the Manager object
func NewManager(ip net.IP, port, capacity int) (*Manager, error) {
	handler, err := NewHandler(capacity)

	if err != nil {
		return nil, err
	}

	return &Manager{
		listenIP:        ip,
		listenPort:      port,
		handler:         handler,
		managedBackends: make(map[string]*managedBackend)}, nil
}

// Listen listens for new connections, registering them if needed
func (m *Manager) Listen() {
	// gonna have to gracefully close this bitch somehow
	var err error
	m.listener, err = net.Listen("tcp", m.listenIP.String()+":"+strconv.Itoa(m.listenPort))
	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			log.Println(err)
		}
		go m.attemptRegister(conn)
	}
}

// Get gets the backend associated with a key
func (m *Manager) Get(key string) (*Backend, error) {
	return m.handler.Get(key)
}

// GetBackends gets all the current backends
func (m *Manager) GetBackends() []*Backend {
	return m.handler.GetBackends()
}

// registration message should be in the following format:
// REGISTER <desired_name> <ip>
func (m *Manager) attemptRegister(conn net.Conn) {
	comm := util.NewTCPCommunicator(conn, READ_TIMEOUT, WRITE_TIMEOUT)

	response, err := comm.ReadLine()
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}

	response = strings.Trim(response, " \n\r\t\f")
	tokens := strings.Split(response, " ")

	if len(tokens) < 3 || tokens[0] != "REGISTER" {
		comm.WriteLine("INVALID registration failed")
		comm.Close()
		return
	}

	cleaner, err := regexp.Compile("[^a-zA-Z0-9-_.]+")
	if err != nil {
		log.Panic(err)
	}
	cleanedName := cleaner.ReplaceAllString(tokens[1], "")

	ip := net.ParseIP(tokens[2])
	if ip == nil {
		comm.WriteLine("INVALID ip not parseable")
		comm.Close()
		return
	}

	err = m.handler.Add(cleanedName, ip)
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}

	comm.WriteLine("REGISTERED " + cleanedName + " " + ip.String())
	back := &managedBackend{
		name:   cleanedName,
		dataIP: ip,
		comm:   comm,
	}

	m.managedBackends[cleanedName] = back

	m.monitor(cleanedName)
}

func (m *Manager) monitor(name string) {
	back := m.managedBackends[name]
	comm := back.comm
	for {
		message, err := comm.ReadLine()
		if err != nil {
			if errChk, ok := err.(net.Error); ok && errChk.Timeout() {
				log.Println(err)
				m.deregisterClient(name, "health check timeout ran out")
			} else {
				log.Println(err)
				m.deregisterClient(name, "error reading from tcp connection: "+err.Error())
			}
			return
		}

		tokens := strings.Split(message, " ")
		if len(tokens) < 1 {
			comm.WriteLine("INVALID empty message")
			continue
		}

		switch tokens[0] {
		case "DEREGISTER":
			m.deregisterClient(name, "client requested deregistration")
			return

		case "PAUSE":
			err := m.handler.Remove(name)
			if err != nil {
				comm.WriteLine("INVALID backend already paused")
			} else {
				comm.WriteLine("PAUSED " + name)
			}

		case "RESUME":
			err := m.handler.Add(name, back.dataIP)
			if err != nil {
				comm.WriteLine("INVALID backend already active")
			} else {
				comm.WriteLine("RESUMED " + name)
			}

		case "HEALTH":
			if len(tokens) < 2 {
				comm.WriteLine("INVALID no status code in health check")
			} else {
				comm.WriteLine("HEALTHACK 200")
			}

		default:
			comm.WriteLine("INVALID first token of message (" + tokens[0] + ") is not an option")
		}
	}
}

func (m *Manager) deregisterClient(name, reason string) {
	m.handler.Remove(name)
	comm := m.managedBackends[name].comm
	comm.WriteLine("DEREGISTERED " + name + " " + reason)
	comm.Close()
}
