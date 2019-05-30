package backends

import (
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const (
	REGISTER_TIMEOUT = 10
	READ_TIMEOUT     = 20
	WRITE_TIMEOUT    = 5
)

// Manager contains the info needed to manage the backends
type Manager struct {
	listener      net.Listener                   // listener for new backends
	listenIP      net.IP                         // ip to listen on
	listenPort    int                            // port to listen on
	handler       *Handler                       // handler for backends
	communicators map[string]BackendCommunicator // map of backend name to its communicator
}

// NewManager instantiates a new instance of the Manager object
func NewManager(ip net.IP, port, capacity int) (*Manager, error) {
	handler, err := NewHandler(capacity)

	if err != nil {
		return nil, err
	}

	return &Manager{
		listenIP:      ip,
		listenPort:    port,
		handler:       handler,
		communicators: make(map[string]BackendCommunicator)}, nil
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
	comm := NewTCPCommunicator(conn, READ_TIMEOUT, WRITE_TIMEOUT)

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
	m.communicators[cleanedName] = comm

	// now do your listening thing
}
