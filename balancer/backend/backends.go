package backend

import (
	"github.com/kkdai/maglev"
)

// BackendHandler contains a maglev hashtable of backends and a mapping from neighbor name
// to device name
type BackendHandler struct {
	backHash *maglev.Maglev
	// hashes from neighbor name to gre tunnel device name
	backends map[string]string
}

// New creates a new BackendHandler
func New(capacity int64) (*BackendHandler, error) {
	mag, err := maglev.NewMaglev([]string{}, uint64(capacity))
	if err != nil {
		return nil, err
	}
	return &BackendHandler{mag, make(map[string]string)}, nil
}

// Get gets the backend device name for a given string. Returns error if something goes wrong in
// maglev hashing process
func (bh *BackendHandler) Get(key string) (string, error) {
	back, err := bh.backHash.Get(key)
	if err != nil {
		return "", err
	}
	return bh.backends[back], nil
}

// Add adds a new backend and its associated GRE device. Throws error if the maglev table is out of
// slots
func (bh *BackendHandler) Add(name string, devName string) error {
	err := bh.backHash.Add(name)
	if err != nil {
		return err
	}
	bh.backends[name] = devName
	return nil
}

// Remove removes an entry from the backends. Throws error if backend does not exist
func (bh *BackendHandler) Remove(name string) error {
	err := bh.backHash.Remove(name)
	if err != nil {
		return err
	}
	delete(bh.backends, name)
	return nil
}
