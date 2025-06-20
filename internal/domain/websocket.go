package domain

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// WebSocketEvent represents a WebSocket event message
type WebSocketEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// CounterData represents counter update data
type CounterData struct {
	Count  int    `json:"count"`
	RoomID string `json:"room_id"`
}

// UserJoinData represents user join data
type UserJoinData struct {
	UserID      string `json:"user_id"`
	RoomID      string `json:"room_id"`
	TotalUsers  int    `json:"total_users"`
	JoinedAt    string `json:"joined_at"`
}

// UserLeaveData represents user leave data
type UserLeaveData struct {
	UserID      string `json:"user_id"`
	RoomID      string `json:"room_id"`
	TotalUsers  int    `json:"total_users"`
	LeftAt      string `json:"left_at"`
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	Conn   *websocket.Conn
	RoomID string
	UserID string
}

// Room represents a WebSocket room
type Room struct {
	ID          string
	Connections map[string]*WebSocketConnection
	Counter     int
	Ticker      *time.Ticker
	StopChan    chan bool
	Started     bool
	Mutex       sync.RWMutex
}

// RoomManager manages all WebSocket rooms
type RoomManager struct {
	Rooms map[string]*Room
	Mutex sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

// GetOrCreateRoom gets an existing room or creates a new one
func (rm *RoomManager) GetOrCreateRoom(roomID string) *Room {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	room, exists := rm.Rooms[roomID]
	if !exists {
		room = &Room{
			ID:          roomID,
			Connections: make(map[string]*WebSocketConnection),
			Counter:     0,
			StopChan:    make(chan bool),
			Started:     false,
		}
		rm.Rooms[roomID] = room
	}
	return room
}

// RemoveRoom removes a room if it's empty
func (rm *RoomManager) RemoveRoom(roomID string) {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	if room, exists := rm.Rooms[roomID]; exists {
		room.Mutex.RLock()
		isEmpty := len(room.Connections) == 0
		room.Mutex.RUnlock()

		if isEmpty {
			room.stopCounter()
			delete(rm.Rooms, roomID)
		}
	}
}

// AddConnection adds a connection to a room
func (r *Room) AddConnection(connID string, conn *WebSocketConnection) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.Connections[connID] = conn

	// Broadcast user join event
	joinData := UserJoinData{
		UserID:     conn.UserID,
		RoomID:     r.ID,
		TotalUsers: len(r.Connections),
		JoinedAt:   time.Now().Format(time.RFC3339),
	}
	r.broadcastEvent("user_join", joinData)

	// Start the counter if this is the first connection
	if len(r.Connections) == 1 && !r.Started {
		r.startCounter()
	}
}

// RemoveConnection removes a connection from a room
func (r *Room) RemoveConnection(connID string) {
	r.Mutex.Lock()
	
	conn, exists := r.Connections[connID]
	if !exists {
		r.Mutex.Unlock()
		return
	}
	
	delete(r.Connections, connID)
	totalUsers := len(r.Connections)
	
	// Broadcast user leave event before unlocking
	leaveData := UserLeaveData{
		UserID:     conn.UserID,
		RoomID:     r.ID,
		TotalUsers: totalUsers,
		LeftAt:     time.Now().Format(time.RFC3339),
	}
	r.broadcastEvent("user_leave", leaveData)
	
	r.Mutex.Unlock()

	// Stop the counter if no connections are left
	if totalUsers == 0 {
		r.stopCounter()
	}
}

// broadcastEvent broadcasts an event to all connections in the room
func (r *Room) broadcastEvent(eventType string, data interface{}) {
	event := WebSocketEvent{
		Event: eventType,
		Data:  data,
	}
	
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}
	
	connections := make([]*WebSocketConnection, 0, len(r.Connections))
	for _, conn := range r.Connections {
		connections = append(connections, conn)
	}
	
	// Broadcast to all connections (unlock before network operations)
	for _, wsConn := range connections {
		err := wsConn.Conn.Write(context.Background(), websocket.MessageText, eventJSON)
		if err != nil {
			// Connection error will be handled by the connection handler
			continue
		}
	}
}

// startCounter starts the counter for the room
func (r *Room) startCounter() {
	r.Started = true
	r.Counter = 0
	r.Ticker = time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-r.Ticker.C:
				r.Mutex.Lock()
				r.Counter++
				counter := r.Counter
				roomID := r.ID
				r.Mutex.Unlock()

				// Broadcast counter update event
				counterData := CounterData{
					Count:  counter,
					RoomID: roomID,
				}
				r.Mutex.RLock()
				r.broadcastEvent("counter_update", counterData)
				r.Mutex.RUnlock()
			case <-r.StopChan:
				return
			}
		}
	}()
}

// stopCounter stops the counter for the room
func (r *Room) stopCounter() {
	if r.Ticker != nil {
		r.Ticker.Stop()
		r.Ticker = nil
		r.Started = false
		r.Counter = 0
		select {
		case <-r.StopChan:
		default:
			close(r.StopChan)
		}
		r.StopChan = make(chan bool)
	}
}
