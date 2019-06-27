package backends

import (
	"net"

	"github.com/kkdai/maglev"
)

// Handler contains a maglev hashtable of backends and a mapping from neighbor name to backend
type Handler struct {
	backHash   *maglev.Maglev      // maglev hash for consistent hashing
	backendMap map[string]*Backend // hash from backend name to backend struct
}

// NewHandler creates a new Handler
func NewHandler(capacity int) (*Handler, error) {
	mag, err := maglev.NewMaglev([]string{}, uint64(capacity))
	if err != nil {
		return nil, err
	}
	return &Handler{mag, make(map[string]*Backend)}, nil
}

// Get gets the backend device name for a given string. Returns error if something goes wrong in
// maglev hashing process
func (bh *Handler) Get(key string) (*Backend, error) {
	back, err := bh.backHash.Get(key)
	if err != nil {
		return nil, err
	}
	return bh.backendMap[back], nil
}

// Add adds a new backend and its associated Backend struct. Throws error if the maglev table is out of
// slots or if the entry already exists
func (bh *Handler) Add(name string, ip net.IP) error {
	err := bh.backHash.Add(name)
	if err != nil {
		return err
	}
	conn, err := net.Dial("udp", ip.String()+":1337")
	backend := NewBackend(name, ip, NewUDPForwarder(conn))
	bh.backendMap[name] = backend
	return nil
}

// Remove removes an entry from the backends. Throws error if backend does not exist
func (bh *Handler) Remove(name string) error {
	err := bh.backHash.Remove(name)
	if err != nil {
		return err
	}
	delete(bh.backendMap, name)
	return nil
}

// GetBackends returns a slice of all the backends
func (bh *Handler) GetBackends() []*Backend {
	toReturn := make([]*Backend, 0, len(bh.backendMap))
	for _, val := range bh.backendMap {
		toReturn = append(toReturn, val)
	}
	return toReturn
}
