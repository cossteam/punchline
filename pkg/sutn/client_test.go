package stunclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSTUNClient(t *testing.T) {
	client, err := NewClient("stun:stun.easyvoip.com:3478")
	assert.NoError(t, err)
	defer client.Close()

	t.Run("XORMappedAddress", func(t *testing.T) {
		addr, err := client.XORMappedAddress()
		assert.NoError(t, err)
		assert.NotNil(t, addr)
		t.Logf("XORMappedAddress: %s", addr)
	})

	t.Run("MappedAddress", func(t *testing.T) {
		addr, err := client.MappedAddress()
		assert.NoError(t, err)
		assert.NotNil(t, addr)
		t.Logf("MappedAddress: %s", addr)
	})

	t.Run("ChangedAddress", func(t *testing.T) {
		addr, err := client.ChangedAddress()
		assert.NoError(t, err)
		assert.NotNil(t, addr)
		t.Logf("ChangedAddress: %s", addr)
	})

	t.Run("OtherAddress", func(t *testing.T) {
		addr, err := client.OtherAddress()
		assert.NoError(t, err)
		assert.NotNil(t, addr)
		t.Logf("OtherAddress: %s", addr)
	})

	t.Run("ExternalAddrs", func(t *testing.T) {
		addrs, err := client.ExternalAddrs()
		assert.NoError(t, err)
		assert.NotNil(t, addrs)
		t.Logf("ExternalAddrs: %s", addrs)
	})
}
