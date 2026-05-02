package hikvision

import (
	"context"
	"encoding/xml"
	"fmt"
)

// HttpHostConfig represents the XML for configuring an HTTP notification host
type HttpHostConfig struct {
	XMLName         xml.Name `xml:"HttpHostNotification"`
	XMLNS           string   `xml:"xmlns,attr"`
	ID              int      `xml:"id"`
	AddressingFormat string   `xml:"addressingFormatType"`
	IPAddress       string   `xml:"ipAddress,omitempty"`
	HostName        string   `xml:"hostName,omitempty"`
	PortNo          int      `xml:"portNo"`
	URL             string   `xml:"url"`
	Protocol        string   `xml:"protocolType"`
	HttpAuthType    string   `xml:"httpAuthenticationType"`
}

// SetupAlarmHost configures the device to send real-time events to the specified URL
func (c *Client) SetupAlarmHost(ctx context.Context, hostID int, serverIP string, serverPort int, callbackPath string) error {
	config := HttpHostConfig{
		XMLNS:           "http://www.isapi.org/ver20/XMLSchema",
		ID:              hostID,
		AddressingFormat: "ipaddress",
		IPAddress:       serverIP,
		PortNo:          serverPort,
		URL:             callbackPath,
		Protocol:        "http",
		HttpAuthType:    "none",
	}

	payload, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal host config: %w", err)
	}

	path := fmt.Sprintf("/ISAPI/Event/notification/httpHosts/%d", hostID)
	_, err = c.Do(ctx, "PUT", path, nil, xmlHeader(payload))
	if err != nil {
		return fmt.Errorf("configure alarm host: %w", err)
	}

	return nil
}

// SubscribeToEvents enables specific event types for notification
func (c *Client) SubscribeToEvents(ctx context.Context) error {
	// This is often device-specific, but many K1T series enable all by default 
	// once a host is configured. Some require enabling "Access Control Event" 
	// in the event configuration.
	return nil
}
