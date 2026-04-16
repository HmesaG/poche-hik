package hikvision

import (
	"context"
	"encoding/xml"
	"fmt"
)

// DoorStatus represents the current status of a door
type DoorStatus struct {
	DoorNo     int    `xml:"doorNo"`
	DoorName   string `xml:"doorName"`
	DoorStatus string `xml:"doorStatus"` // closed, open
	LockStatus string `xml:"lockStatus"` // locked, unlocked
}

// RemoteOpen opens a door remotely
func (c *Client) RemoteOpen(ctx context.Context, doorNo int) error {
	path := fmt.Sprintf("/ISAPI/AccessControl/Door/%d/RemoteOpen", doorNo)
	_, err := c.Do(ctx, "PUT", path, nil, nil)
	return err
}

// RemoteClose closes a door remotely (if supported)
func (c *Client) RemoteClose(ctx context.Context, doorNo int) error {
	path := fmt.Sprintf("/ISAPI/AccessControl/Door/%d/RemoteClose", doorNo)
	_, err := c.Do(ctx, "PUT", path, nil, nil)
	return err
}

// GetDoorStatus retrieves the status of a specific door
func (c *Client) GetDoorStatus(ctx context.Context, doorNo int) (*DoorStatus, error) {
	path := fmt.Sprintf("/ISAPI/AccessControl/Door/status/%d", doorNo)
	resp, err := c.Do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var status DoorStatus
	if err := xml.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("unmarshal door status: %w", err)
	}

	return &status, nil
}
