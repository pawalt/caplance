package backend

import (
	"errors"
	"github.com/kkdai/maglev"
	"math/big"
	"net"
)

type BackendHandler struct {
	backHash *maglev.Maglev
	backends map[string]net.IP
}

func New(capacity int64) (*BackendHandler, error) {
	if !big.NewInt(capacity).ProbablyPrime(10) {
		return nil, errors.New("Capacity not prime")
	}
	return &BackendHandler{maglev.NewMaglev([]string{}, uint64(capacity)), make(map[string]net.IP)}, nil
}

func (bh *BackendHandler) Get(key string) (net.IP, error) {
	back, err := bh.backHash.Get(key)
	if err != nil {
		return nil, err
	}
	return bh.backends[back], nil
}

func (bh *BackendHandler) Add(name string, ip net.IP) error {
	err := bh.backHash.Add(name)
	if err != nil {
		return err
	}
	bh.backends[name] = ip
	return nil
}

func (bh *BackendHandler) Remove(name string) error {
	err := bh.backHash.Remove(name)
	if err != nil {
		return err
	}
	delete(bh.backends, name)
	return nil
}
