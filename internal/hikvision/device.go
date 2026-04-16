package hikvision

import (
	"context"
	"encoding/xml"
	"fmt"
)

// DeviceInfo represents the response from /ISAPI/System/deviceInfo
type DeviceInfo struct {
	XMLName           xml.Name `xml:"DeviceInfo"`
	DeviceName        string   `xml:"deviceName"`
	DeviceID          string   `xml:"deviceID"`
	Model             string   `xml:"model"`
	SerialNumber      string   `xml:"serialNumber"`
	MACAddress        string   `xml:"macAddress"`
	FirmwareVersion   string   `xml:"firmwareVersion"`
	FirmwareReleasedDate string `xml:"firmwareReleasedDate"`
	DeviceType        string   `xml:"deviceType"`
}

// GetDeviceInfo retrieves static information about the device
func (c *Client) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	resp, err := c.Do(ctx, "GET", "/ISAPI/System/deviceInfo", nil, nil)
	if err != nil {
		return nil, err
	}

	var info DeviceInfo
	if err := xml.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("unmarshal device info: %w (body: %s)", err, string(resp))
	}

	return &info, nil
}

// Reboot reboots the device
func (c *Client) Reboot(ctx context.Context) error {
	_, err := c.Do(ctx, "PUT", "/ISAPI/System/reboot", nil, nil)
	return err
}
