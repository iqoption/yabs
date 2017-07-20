package base

import (
	"github.com/bradfitz/gomemcache/memcache"
)

type Memcache struct {
	client *memcache.Client
}

func (m *Memcache) Get(key string) (string, error) {
	item, err := m.client.Get(key)
	if err != nil {
		return "", err
	}

	return string(item.Value), nil
}

func (m *Memcache) Set(key, value string) error {
	err := m.client.Set(&memcache.Item{Key: key, Value: []byte(value)})
	return err
}

func NewMemcache(servers []string) (*Memcache, error) {
	return &Memcache{memcache.New(servers...)}, nil
}
