package chat

import (
	"sort"
	"sync"
)

type HubManager struct {
	hubs map[string]*Hub
	mu   sync.RWMutex
}

type HubInfo struct {
	name  string
	count int
}

func NewRoomManager() *HubManager {
	return &HubManager{
		hubs: make(map[string]*Hub),
	}
}

// create (to do: fix the case where two users want the same room name)
func (m *HubManager) GetOrCreate(name string) *Hub {
	m.mu.RLock()
	hub, ok := m.hubs[name]
	m.mu.RUnlock()
	if ok {
		return hub
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if hub, ok := m.hubs[name]; ok { // re-check if other go routine created the room
		return hub
	}
	h := NewHub(name)
	m.hubs[name] = h
	return h
}

// list
func (m *HubManager) ListRooms() []HubInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hubs := make([]HubInfo, 0, len(m.hubs))
	for name, hub := range m.hubs {
		hubs = append(hubs, HubInfo{name, hub.UserCount()})
	}
	sort.Slice(hubs, func(i, j int) bool {
		return hubs[i].name < hubs[j].name
	})
	return hubs
}

// find
func (m *HubManager) Find(name string) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[name]
}

// delete (to do: handle users still in a room)
func (m *HubManager) Delete(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hubs, name)
}
