package api

import (
	"encoding/xml"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"ponches/internal/store"
)

// HikvisionEventNotification represents the XML payload sent by the device
type HikvisionEventNotification struct {
	XMLName    xml.Name `xml:"EventNotificationAlert"`
	ID         string   `xml:"id"`
	IPAddress  string   `xml:"ipAddress"`
	PortNo     int      `xml:"portNo"`
	Protocol   string   `xml:"protocolType"`
	MacAddress string   `xml:"macAddress"`
	DateTime   string   `xml:"dateTime"`
	ActivePost int      `xml:"activePostCount"`
	EventType  string   `xml:"eventType"`
	EventState string   `xml:"eventState"`
	EventDescription string `xml:"eventDescription"`
	
	// Access Control specific fields
	AccessControllerEvent struct {
		EmployeeNo string `xml:"employeeNoString"`
		EventLogID uint32 `xml:"eventLogID"`
		Major      uint32 `xml:"major"`
		Minor      uint32 `xml:"minor"`
		CardNo     string `xml:"cardNo"`
	} `xml:"AccessControllerEvent"`
}

func (s *Server) handleHikvisionCallback(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read Hikvision callback body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var event HikvisionEventNotification
	if err := xml.Unmarshal(body, &event); err != nil {
		// Try JSON if XML fails (some newer devices)
		log.Trace().Msg("Failed to unmarshal Hikvision XML callback, ignoring or logging raw body")
		// For now we assume XML as per standard ISAPI
		w.WriteHeader(http.StatusOK) // Device expects 200 even if we don't process it
		return
	}

	// We only care about Access Control events with an EmployeeNo
	empNo := event.AccessControllerEvent.EmployeeNo
	if empNo == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Info().
		Str("employeeNo", empNo).
		Str("eventType", event.EventType).
		Str("device", event.IPAddress).
		Msg("Received real-time event from Hikvision device")

	// Parse timestamp
	ts, err := time.Parse("2006-01-02T15:04:05-07:00", event.DateTime)
	if err != nil {
		ts = time.Now()
	}

	// Save event
	attEvent := &store.AttendanceEvent{
		ID:         uuid.New().String(),
		DeviceID:   event.IPAddress,
		EmployeeNo: empNo,
		Timestamp:  ts,
		Type:       "check-in", // We default to check-in, the engine will handle pairs
	}

	if err := s.Store.SaveEvent(r.Context(), attEvent); err != nil {
		log.Error().Err(err).Str("empNo", empNo).Msg("Failed to save real-time event")
	}

	// Fetch employee name for the real-time broadcast
	empName := empNo
	emp, _ := s.Store.GetEmployeeByNo(r.Context(), empNo)
	if emp != nil {
		empName = emp.FirstName + " " + emp.LastName
	}

	// Broadcast to real-time dashboard listeners (WebSockets)
	s.Hub.BroadcastAttendanceEvent(empNo, empName, event.IPAddress, ts)

	w.WriteHeader(http.StatusOK)
}
