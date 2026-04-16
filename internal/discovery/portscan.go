package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ScanPorts scans for potential Hikvision devices by checking common ISAPI ports.
// It can scan a specific CIDR range or auto-detect based on local interfaces if range is empty.
func ScanPorts(ctx context.Context, rangeCIDR string, ports []int, timeoutPerHost time.Duration, maxConcurrency int) ([]string, error) {
	log.Debug().Str("range", rangeCIDR).Ints("ports", ports).Msg("Starting TCP port scan")

	var targets []string
	if rangeCIDR != "" {
		hosts, err := getHostsFromCIDR(rangeCIDR)
		if err != nil {
			return nil, fmt.Errorf("parse CIDR: %w", err)
		}
		targets = hosts
	} else {
		// Auto-detect targets from local /24 subnets
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, fmt.Errorf("list interfaces: %w", err)
		}

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
				targets = append(targets, getSubnetHosts(ipnet)...)
			}
		}
	}

	if len(targets) == 0 {
		return []string{}, nil
	}

	log.Debug().Int("hosts", len(targets)).Msg("Targets identified for port scanning")

	results := make(chan string)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)

	// Context for cancellation
	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for _, host := range targets {
			for _, port := range ports {
				select {
				case <-scanCtx.Done():
					return
				default:
					wg.Add(1)
					go func(h string, p int) {
						defer wg.Done()
						select {
						case semaphore <- struct{}{}:
							defer func() { <-semaphore }()
						case <-scanCtx.Done():
							return
						}

						address := fmt.Sprintf("%s:%d", h, p)
						conn, err := net.DialTimeout("tcp", address, timeoutPerHost)
						if err == nil {
							conn.Close()
							results <- h
						}
					}(host, port)
				}
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

func getHostsFromCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var hosts []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		hosts = append(hosts, ip.String())
	}

	// Remove network and broadcast addresses if it's a typical subnet
	if len(hosts) > 2 {
		return hosts[1 : len(hosts)-1], nil
	}
	return hosts, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func getSubnetHosts(ipnet *net.IPNet) []string {
	var hosts []string
	ip := ipnet.IP.Mask(ipnet.Mask)
	
	ones, bits := ipnet.Mask.Size()
	// Limit auto-discovery to /24 to avoid huge scans on misconfigured systems
	if ones < 24 {
		ones = 24
	}

	numHosts := 1 << (bits - ones)
	for i := 1; i < numHosts-1; i++ {
		hostIP := make(net.IP, len(ip))
		copy(hostIP, ip)
		
		// This assumes IPv4 for simplicity in this specific project
		hostIP[len(hostIP)-1] = byte(i)
		hosts = append(hosts, hostIP.String())
	}
	return hosts
}
