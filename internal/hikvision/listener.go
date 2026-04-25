package hikvision

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// EventListener listens for real-time events from Hikvision devices.
// It shares the authenticated Client so all requests use Digest Auth.
type EventListener struct {
	client   *Client
	handlers []EventHandler
	mu       sync.RWMutex
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
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

// NewEventListener creates a new event listener backed by a shared ISAPI Client.
func NewEventListener(deviceIP string, port int, username, password string) *EventListener {
	return &EventListener{
		client:   NewClient(deviceIP, port, username, password),
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
			events, err := l.client.GetRecentEvents(l.ctx)
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

// GetEventsInRange fetches attendance events from the device via ISAPI for a specific time range.
func (c *Client) GetEventsInRange(ctx context.Context, start, end time.Time) ([]*AttendanceEvent, error) {
	// For this device, JSON is required and strict about the time format.
	// We'll use the format: 2026-04-25T17:53:41+08:00
	// We'll try to detect the offset or use +08:00 as a fallback.
	offset := "+08:00" 
	
	reqBody := map[string]interface{}{
		"AcsEventCond": map[string]interface{}{
			"searchID":             fmt.Sprintf("search-%d", time.Now().UnixMilli()),
			"searchResultPosition": 0,
			"maxResults":           1000,
			"major":                0,
			"minor":                0,
			"startTime":           start.Format("2006-01-02T15:04:05") + offset,
			"endTime":             end.Format("2006-01-02T15:04:05") + offset,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := c.Do(ctx, "POST", "/ISAPI/AccessControl/AcsEvent?format=json", 
		map[string]string{"Content-Type": "application/json"}, body)
	
	if err != nil {
		// Fallback to XML if JSON is not supported (older devices)
		log.Debug().Err(err).Msg("JSON AcsEvent failed, trying XML fallback")
		return c.getEventsInRangeXML(ctx, start, end)
	}

	// Parse JSON Response
	type acsEvent struct {
		EmployeeNo       string `json:"employeeNo"`
		EmployeeNoString string `json:"employeeNoString"`
		Time             string `json:"time"`
		Major            int    `json:"major"`
		Minor            int    `json:"minor"`
	}
	type acsEventRes struct {
		AcsEvent struct {
			TotalMatches int        `json:"totalMatches"`
			InfoList     []acsEvent `json:"InfoList"`
		} `json:"AcsEvent"`
	}

	var res acsEventRes
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, fmt.Errorf("unmarshal JSON AcsEvent: %w", err)
	}

	var events []*AttendanceEvent
	for _, info := range res.AcsEvent.InfoList {
		empNo := info.EmployeeNoString
		if empNo == "" {
			empNo = info.EmployeeNo
		}
		if empNo == "" {
			continue
		}

		// We ignore the timezone offset from the device and treat it as local wall clock time.
		// This is necessary because devices often have a different timezone (+08:00) than the server,
		// but the wall clocks are synchronized.
		timePart := info.Time
		if len(timePart) > 19 {
			timePart = timePart[:19]
		}
		t, _ := time.ParseInLocation("2006-01-02T15:04:05", strings.Replace(timePart, " ", "T", 1), time.Local)
		if t.IsZero() {
			t, _ = time.ParseInLocation("2006-01-02 15:04:05", timePart, time.Local)
		}

		events = append(events, &AttendanceEvent{
			EmployeeNo:    empNo,
			DeviceID:      c.Host,
			Timestamp:     t,
			EventType:     fmt.Sprintf("Major%d-Minor%d", info.Major, info.Minor),
			Verified:      true,
			AccessGranted: true,
		})
	}

	return events, nil
}

func (c *Client) getEventsInRangeXML(ctx context.Context, start, end time.Time) ([]*AttendanceEvent, error) {
	xmlBody := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<AcsEventCond xmlns="http://www.isapi.org/ver20/XMLSchema">
	<searchID>%s</searchID>
	<searchResultPosition>0</searchResultPosition>
	<maxResults>1000</maxResults>
	<major>0</major>
	<minor>0</minor>
	<startTime>%s</startTime>
	<endTime>%s</endTime>
</AcsEventCond>`, 
		uuid(), 
		start.Format("2006-01-02T15:04:05Z"),
		end.Format("2006-01-02T15:04:05Z"))

	resp, err := c.Do(ctx, "POST", "/ISAPI/AccessControl/AcsEvent", 
		map[string]string{"Content-Type": "application/xml"}, []byte(xmlBody))
	if err != nil {
		return nil, err
	}

	type acsEvent struct {
		EmployeeNo string `xml:"employeeNo"`
		Time       string `xml:"time"`
		Major      int    `xml:"major"`
		Minor      int    `xml:"minor"`
	}
	type acsEventRes struct {
		XMLName      xml.Name   `xml:"AcsEvent"`
		TotalMatches int        `xml:"totalMatches"`
		InfoList     []acsEvent `xml:"InfoList>AcsEvent"`
	}

	var res acsEventRes
	if err := xml.Unmarshal(resp, &res); err != nil {
		return nil, err
	}

	var events []*AttendanceEvent
	for _, info := range res.InfoList {
		if info.EmployeeNo == "" {
			continue
		}
		timePart := info.Time
		if len(timePart) > 19 {
			timePart = timePart[:19]
		}
		t, _ := time.ParseInLocation("2006-01-02T15:04:05", strings.Replace(timePart, " ", "T", 1), time.Local)
		if t.IsZero() {
			t, _ = time.ParseInLocation("2006-01-02 15:04:05", timePart, time.Local)
		}

		events = append(events, &AttendanceEvent{
			EmployeeNo:    info.EmployeeNo,
			DeviceID:      c.Host,
			Timestamp:     t,
			EventType:     fmt.Sprintf("Major%d-Minor%d", info.Major, info.Minor),
			Verified:      true,
			AccessGranted: true,
		})
	}
	return events, nil
}

// GetRecentEvents fetches recent attendance events (last 24h) from the device.
func (c *Client) GetRecentEvents(ctx context.Context) ([]*AttendanceEvent, error) {
	return c.GetEventsInRange(ctx, time.Now().Add(-24*time.Hour), time.Now().Add(1*time.Hour))
}

func uuid() string {
	_, _ =  time.Now().UnixNano(), 0 // dummy
	// For searchID, a simple string is often enough for Hikvision
	return fmt.Sprintf("search-%d", time.Now().UnixMilli())
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
