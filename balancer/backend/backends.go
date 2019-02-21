package backend

import (
	"errors"
	"github.com/kkdai/maglev"
	"math/big"
)

type BackendHandler struct {
	backHash *maglev.Maglev
	// hashes from neighbor name to gre tunnel device name
	backends map[string]string
}

func New(capacity int64) (*BackendHandler, error) {
	if !big.NewInt(capacity).ProbablyPrime(10) {
		return nil, errors.New("Capacity not prime")
	}
	return &BackendHandler{maglev.NewMaglev([]string{}, uint64(capacity)), make(map[string]string)}, nil
}

func (bh *BackendHandler) Get(key string) (string, error) {
	back, err := bh.backHash.Get(key)
	if err != nil {
		return "", err
	}
	return bh.backends[back], nil
}

func (bh *BackendHandler) Add(name string, devName string) error {
	err := bh.backHash.Add(name)
	if err != nil {
		return err
	}
	bh.backends[name] = devName
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
