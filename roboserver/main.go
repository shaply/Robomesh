package main

import (
	"fmt"
	"net"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from RoboHub!")
}

func getLocalIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces that are down
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip IPv6 and loopback addresses
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			ips = append(ips, ip.String())
		}
	}

	return ips
}

func main() {
	fmt.Println("Starting RoboHub server on port 8080...")

	// Print IP addresses
	ips := getLocalIPs()
	if len(ips) > 0 {
		fmt.Println("Server available at:")
		for _, ip := range ips {
			fmt.Printf("http://%s:8080\n", ip)
		}
	} else {
		fmt.Println("Could not determine server IP addresses")
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
