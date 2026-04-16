package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ScanPorts scans for potential Hikvision devices by checking common ISAPI ports
// in the local subnets of the provided network interfaces.
func ScanPorts(ctx context.Context, ports []int, timeoutPerHost time.Duration) ([]string, error) {
	log.Info().Msg("Starting TCP port scan for Hikvision devices")

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}

	var targets []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}

			// Simple scanner for the current /24 subnet for speed
			// In a real enterprise app, we might want to scan the full range
			// but for this project /24 of each interface is a good start.
			targets = append(targets, getSubnetHosts(ipnet)...)
		}
	}

	log.Info().Int("hosts", len(targets)).Msg("Targets identified for port scanning")

	results := make(chan string)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 100) // Limit concurrency

	go func() {
		for _, host := range targets {
			for _, port := range ports {
				wg.Add(1)
				go func(h string, p int) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					address := fmt.Sprintf("%s:%d", h, p)
					conn, err := net.DialTimeout("tcp", address, timeoutPerHost)
					if err == nil {
						conn.Close()
						// Check if it's likely a Hikvision device would require
						// an ISAPI call, but for now we just return the open port.
						results <- h
					}
				}(host, port)
			}
		}
		wg.Wait()
		close(results)
	}()

	uniqueHosts := make(map[string]struct{})
	for host := range results {
		uniqueHosts[host] = struct{}{}
	}

	var finalHosts []string
	for h := range uniqueHosts {
		finalHosts = append(finalHosts, h)
	}

	log.Info().Int("found", len(finalHosts)).Msg("TCP port scan completed")
	return finalHosts, nil
}

func getSubnetHosts(ipnet *net.IPNet) []string {
	var hosts []string
	ip := ipnet.IP.Mask(ipnet.Mask)
	
	// Only scan /24 or smaller networks to avoid massive wait times
	ones, bits := ipnet.Mask.Size()
	if ones < 24 {
		// If network is larger than /24, just scan the .0/24 part of that network for safety
		// Real enterprise scanners would be more thorough
		ones = 24
	}

	numHosts := 1 << (bits - ones)
	for i := 1; i < numHosts-1; i++ {
		hostIP := make(net.IP, len(ip))
		copy(hostIP, ip)
		
		// Add offset to the last byte for /24
		hostIP[3] = byte(i)
		hosts = append(hosts, hostIP.String())
	}
	return hosts
}
