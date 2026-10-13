package chat

import (
	"fmt"
	"nite-chat/internal/chat/message"
	"sort"
	"sync"
)

type HubManager struct {
	hubs map[string]*Hub
	mu   sync.RWMutex
}

type HubInfo = message.HubInfo

func NewHubManager() *HubManager {
	return &HubManager{
		hubs: make(map[string]*Hub),
	}
}

// create (to do: fix the case where two users want the same room name)
func (m *HubManager) GetOrCreate(name string) message.Hub {
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
	h := NewHub(name, func(hub *Hub) {
		m.deleteIfEmpty(name, hub)
	})
	m.hubs[name] = h
	go h.Run()
	fmt.Printf("Created hub: %s\n", name)
	return h
}

// list
func (m *HubManager) ListRooms() []HubInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hubs := make([]HubInfo, 0, len(m.hubs))
	for _, hub := range m.hubs {
		name, userCount := hub.HubDetails()
		hubs = append(hubs, HubInfo{Name: name, Count: userCount})
	}
	sort.Slice(hubs, func(i, j int) bool {
		return hubs[i].Name < hubs[j].Name
	})
	return hubs
}

func (m * HubManager) GetRoomInfo(name string) (HubInfo, bool) {
	m.mu.RLock()
	hub, ok := m.hubs[name]
	m.mu.RUnlock()
	if !ok {
		return message.HubInfo{}, false
	}

	hubName, userCount := hub.HubDetails()
	return message.HubInfo{Name: hubName, Count: userCount}, true
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
	hub := m.hubs[name]
	if hub != nil {
		hub.Shutdown()
	}

	delete(m.hubs, name)
}

func (m *HubManager) deleteIfEmpty(name string, hub *Hub) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hubs[name] != hub {
		return
	}

	hub.Shutdown()
	delete(m.hubs, name)
}
