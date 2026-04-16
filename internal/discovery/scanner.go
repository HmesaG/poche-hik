package discovery

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	sadpMulticastIP   = "239.255.255.250"
	sadpMulticastPort = 37020
)

// SADPProbe defines the XML structure for the UDP multicast probe
type SADPProbe struct {
	XMLName xml.Name `xml:"Probe"`
	Uuid    string   `xml:"Uuid"`
	Types   string   `xml:"Types"`
}

// SADPDevice represents a Hikvision device discovered via SADP
type SADPDevice struct {
	UUID          string `xml:"Uuid"`
	Types         string `xml:"Types"`
	DeviceType    string `xml:"DeviceType"`
	DeviceDesc    string `xml:"DeviceDescription"`
	DeviceSN      string `xml:"DeviceSN"`
	MAC           string `xml:"MAC"`
	IPv4Address   string `xml:"IPv4Address"`
	IPv4Subnet    string `xml:"IPv4SubnetMask"`
	IPv4Gateway   string `xml:"IPv4Gateway"`
	IPv6Address   string `xml:"IPv6Address"`
	IPv6Gateway   string `xml:"IPv6Gateway"`
	IPv6MaskLen   int    `xml:"IPv6MaskLen"`
	CommandPort   int    `xml:"CommandPort"`
	HttpPort      int    `xml:"HttpPort"`
	DHCP          string `xml:"DHCP"`
	DSPVersion    string `xml:"DSPVersion"`
	Activated     bool   `xml:"Activated"`
	PasswordReset string `xml:"PasswordResetAbility"`
}

// Discover performs a SADP multicast discovery and returns a list of unique devices found
func Discover(timeoutSeconds int) ([]SADPDevice, error) {
	log.Info().Msgf("Starting aggressive SADP discovery (timeout: %ds)", timeoutSeconds)

	devices := make(map[string]SADPDevice)
	var mu sync.Mutex

	// Prepare probe
	probeUUID := strings.ToUpper(uuid.New().String())
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?><Probe><Uuid>`)
	buf.WriteString(probeUUID)
	buf.WriteString(`</Uuid><Types>inquiry</Types></Probe>`)
	probeData := buf.Bytes()

	multicastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", sadpMulticastIP, sadpMulticastPort))
	if err != nil {
		return nil, err
	}

	// Listen on all interfaces
	listenAddr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	listenConn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	defer listenConn.Close()

	stopRead := make(chan bool)
	go func() {
		buffer := make([]byte, 8192)
		for {
			select {
			case <-stopRead:
				return
			default:
				listenConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				n, _, err := listenConn.ReadFromUDP(buffer)
				if err != nil {
					continue
				}

				var dev SADPDevice
				if err := xml.Unmarshal(buffer[:n], &dev); err == nil && dev.MAC != "" {
					mu.Lock()
					devices[dev.MAC] = dev
					mu.Unlock()
					log.Debug().Str("ip", dev.IPv4Address).Str("mac", dev.MAC).Msg("Found device via SADP")
				}
			}
		}
	}()

	// Iterate over interfaces to send probes
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}

			localAddr, _ := net.ResolveUDPAddr("udp", ipnet.IP.String()+":0")
			conn, err := net.ListenUDP("udp", localAddr)
			if err != nil {
				continue
			}

			// Send multiple probes per IP
			for i := 0; i < 2; i++ {
				conn.WriteToUDP(probeData, multicastAddr)
				time.Sleep(20 * time.Millisecond)
			}
			conn.Close()
		}
	}

	time.Sleep(time.Duration(timeoutSeconds) * time.Second)
	close(stopRead)

	mu.Lock()
	defer mu.Unlock()
	var result []SADPDevice
	for _, v := range devices {
		result = append(result, v)
	}
	log.Info().Int("count", len(result)).Msg("Aggressive SADP scan finished")
	return result, nil
}
