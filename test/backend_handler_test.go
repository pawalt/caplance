package test

import (
	"testing"

	"github.com/pwpon500/caplance/balancer/backends"
)

func TestNonPrimeCapacity(t *testing.T) {
	_, err := backends.NewHandler(10)
	assert(t, err != nil, "error thrown for non-prime capacity")
}

func TestSingleBackend(t *testing.T) {
	back, err := backends.NewHandler(3)
	ok(t, err)

	back.Add("b1", "gre0")
	expected := "gre0"
	actual, err := back.Get("10.0.0.2:53686")
	ok(t, err)

	equals(t, expected, actual)
}

func TestBackendRemove(t *testing.T) {
	back, err := backends.NewHandler(3)
	ok(t, err)

	m := make(map[string]string)
	m["gre0"] = "b1"
	m["gre1"] = "b2"

	back.Add("b1", "gre0")
	back.Add("b2", "gre1")
	expected, err := back.Get("192.168.1.2:789")
	ok(t, err)

	expectedName := m[expected]
	back.Remove(expectedName)
	actual, err := back.Get("192.168.1.2:789")
	ok(t, err)

	assert(t, expected != actual, "removed backend was not used")
}
