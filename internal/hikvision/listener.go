package hikvision

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EventListener listens for real-time events from Hikvision devices
type EventListener struct {
	deviceIP   string
	port       int
	username   string
	password   string
	client     *http.Client
	handlers   []EventHandler
	mu         sync.RWMutex
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// EventHandler is a callback function for handling events
type EventHandler func(event *AttendanceEvent)

// AttendanceEvent represents an attendance event from the device
type AttendanceEvent struct {
	EmployeeNo  string    `json:"employeeNo"`
	DeviceID    string    `json:"deviceId"`
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"eventType"` // CardSwipe, Face, Fingerprint, etc.
	DoorID      string    `json:"doorId,omitempty"`
	CardNo      string    `json:"cardNo,omitempty"`
	Verified    bool      `json:"verified"`
	AccessGranted bool  `json:"accessGranted"`
}

// NewEventListener creates a new event listener
func NewEventListener(deviceIP string, port int, username, password string) *EventListener {
	return &EventListener{
		deviceIP: deviceIP,
		port:     port,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		handlers: make([]EventHandler, 0),
	}
}

// AddHandler adds an event handler callback
func (l *EventListener) AddHandler(handler EventHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handler)
}

// Start begins listening for events
func (l *EventListener) Start() error {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return fmt.Errorf("listener already running")
	}
	l.running = true
	l.ctx, l.cancel = context.WithCancel(context.Background())
	l.mu.Unlock()

	go l.pollEvents()
	return nil
}

// Stop stops the listener
func (l *EventListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if !l.running {
		return
	}
	
	l.running = false
	if l.cancel != nil {
		l.cancel()
	}
}

// pollEvents continuously polls the device for new events
func (l *EventListener) pollEvents() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			events, err := l.GetRecentEvents()
			if err != nil {
				continue
			}

			for _, event := range events {
				l.notifyHandlers(event)
			}
		}
	}
}

// notifyHandlers calls all registered handlers with the event
func (l *EventListener) notifyHandlers(event *AttendanceEvent) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, handler := range l.handlers {
		go handler(event)
	}
}

// GetRecentEvents fetches recent events from the device
func (l *EventListener) GetRecentEvents() ([]*AttendanceEvent, error) {
	ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancel()

	// Use ISAPI to get attendance logs
	// This is a simplified implementation - real implementation would use proper ISAPI endpoints
	url := fmt.Sprintf("http://%s:%d/ISAPI/AccessControl/AttendenceLog?format=json", l.deviceIP, l.port)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add digest auth header (simplified - would need proper implementation)
	req.SetBasicAuth(l.username, l.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		AttendenceLogData struct {
			AttendenceLog []struct {
				EmployeeNo  string `json:"employeeNo"`
				VerifyMode  string `json:"verifyMode"`
				DoorID      string `json:"doorID"`
				EventType   string `json:"eventType"`
				Timestamp   string `json:"time"`
			} `json:"AttendenceLog"`
		} `json:"AttendenceLogData"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var events []*AttendanceEvent
	for _, log := range result.AttendenceLogData.AttendenceLog {
		ts, _ := time.Parse(time.RFC3339, log.Timestamp)
		events = append(events, &AttendanceEvent{
			EmployeeNo:  log.EmployeeNo,
			DeviceID:    l.deviceIP,
			Timestamp:   ts,
			EventType:   log.EventType,
			DoorID:      log.DoorID,
			Verified:    true,
			AccessGranted: true,
		})
	}

	return events, nil
}

// PushListener receives push notifications from Hikvision devices via HTTP
type PushListener struct {
	server   *http.Server
	handlers []EventHandler
	mu       sync.RWMutex
}

// NewPushListener creates a new push listener server
func NewPushListener(addr string) *PushListener {
	pl := &PushListener{
		handlers: make([]EventHandler, 0),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ISAPI/Intelligent/Push", pl.handlePush)
	mux.HandleFunc("/event", pl.handlePush)

	pl.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return pl
}

// AddHandler adds an event handler
func (pl *PushListener) AddHandler(handler EventHandler) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.handlers = append(pl.handlers, handler)
}

// Start starts the push listener server
func (pl *PushListener) Start() error {
	return pl.server.ListenAndServe()
}

// Stop stops the push listener server
func (pl *PushListener) Stop() error {
	return pl.server.Shutdown(context.Background())
}

func (pl *PushListener) handlePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event PushEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		// Try XML
		if err := xml.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid event format", http.StatusBadRequest)
			return
		}
	}

	// Convert to AttendanceEvent
	attendanceEvent := &AttendanceEvent{
		EmployeeNo:    event.EmployeeNo,
		DeviceID:      event.DeviceInfo.IPAddress,
		Timestamp:     event.Time,
		EventType:     event.EventType,
		Verified:      event.VerifyResult == 1,
		AccessGranted: event.AccessResult == 1,
	}

	pl.mu.RLock()
	for _, handler := range pl.handlers {
		go handler(attendanceEvent)
	}
	pl.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
}

// PushEvent represents a push notification event from Hikvision
type PushEvent struct {
	XMLName      xml.Name `json:"-" xml:"Event"`
	EventType    string   `json:"eventType" xml:"eventType"`
	EmployeeNo   string   `json:"employeeNo" xml:"employeeNo"`
	DeviceInfo   struct {
		IPAddress string `json:"ipAddress" xml:"ipAddress"`
		DeviceID  string `json:"deviceID" xml:"deviceID"`
	} `json:"deviceInfo" xml:"deviceInfo"`
	Time         time.Time `json:"time" xml:"time"`
	VerifyResult int       `json:"verifyResult" xml:"verifyResult"`
	AccessResult int       `json:"accessResult" xml:"accessResult"`
	DoorID       string    `json:"doorID" xml:"doorID"`
	CardNo       string    `json:"cardNo" xml:"cardNo"`
}
