package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
)

var addressMap = make(map[string]string)

func main() {
	loadEnvVariables()

	udpPort, tcpPort := getPorts()

	checkAddressMapping()

	server := &dns.Server{
		Addr: fmt.Sprintf(":%d", udpPort),
		Net:  "udp",
	}

	dns.HandleFunc(".", handleRequest)

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("UDP server failed to start: %s", err)
		}
	}()

	if tcpPort != 0 {
		tcpServer := &dns.Server{
			Addr: fmt.Sprintf(":%d", tcpPort),
			Net:  "tcp",
		}

		go func() {
			err := tcpServer.ListenAndServe()
			if err != nil {
				log.Fatalf("TCP server failed to start: %s", err)
			}
		}()
	}

	printServerInfo(udpPort, tcpPort)
}

func loadEnvVariables() {
	for _, env := range os.Environ() {
		parts := strings.Split(env, "=")
		key := parts[0]
		if strings.HasPrefix(key, "DNS_MAP") {
			hostname := strings.TrimPrefix(key, "DNS_MAP_")
			address := parts[1]
			addressMap[hostname] = address
		}
	}

	if _, exists := addressMap["conntest.nintendowifi.net"]; !exists {
		defaultAddress := os.Getenv("DNS_DEFAULT_ADDRESS")
		if defaultAddress == "" {
			fmt.Println(color.RedString("Mapping for conntest.nintendowifi.net not found and no default address set."))
			os.Exit(1)
		}
		addressMap["conntest.nintendowifi.net"] = defaultAddress
	}

	if _, exists := addressMap["account.nintendo.net"]; !exists {
		defaultAddress := os.Getenv("DNS_DEFAULT_ADDRESS")
		if defaultAddress == "" {
			fmt.Println(color.RedString("Mapping for account.nintendo.net not found and no default address set."))
			os.Exit(1)
		}
		addressMap["account.nintendo.net"] = defaultAddress
	}
}

func getPorts() (int, int) {
	udpPort := 0
	tcpPort := 0

	if port, exists := os.LookupEnv("UDP_PORT"); exists {
		var err error
		udpPort, err = strconv.Atoi(port)
		if err != nil {
			fmt.Println(color.RedString("Invalid UDP port"))
			os.Exit(1)
		}
	}

	if port, exists := os.LookupEnv("TCP_PORT"); exists {
		var err error
		tcpPort, err = strconv.Atoi(port)
		if err != nil {
			fmt.Println(color.RedString("Invalid TCP port"))
			os.Exit(1)
		}
	}

	if udpPort == 0 && tcpPort == 0 {
		fmt.Println(color.RedString("No server port set. Set one of UDP_PORT or TCP_PORT"))
		os.Exit(1)
	}

	if udpPort == tcpPort {
		fmt.Println(color.RedString("UDP and TCP ports cannot match"))
		os.Exit(1)
	}

	if udpPort == 0 {
		fmt.Println(color.YellowString("UDP port not set. One will be randomly assigned"))
	}

	if tcpPort == 0 {
		fmt.Println(color.YellowString("TCP port not set. One will be randomly assigned"))
	}

	return udpPort, tcpPort
}

func checkAddressMapping() {
	if _, exists := addressMap["conntest.nintendowifi.net"]; !exists {
		fmt.Println(color.RedString("Mapping for conntest.nintendowifi.net not found"))
		os.Exit(1)
	}

	if _, exists := addressMap["account.nintendo.net"]; !exists {
		fmt.Println(color.RedString("Mapping for account.nintendo.net not found"))
		os.Exit(1)
	}
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	question := r.Question[0]
	name := question.Name

	if address, exists := addressMap[name]; exists {
		response := dns.Msg{}
		response.SetReply(r)
		response.Answer = append(response.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: net.ParseIP(address),
		})

		w.WriteMsg(&response)
	}
}

func printServerInfo(udpPort, tcpPort int) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Protocol", "Address"})

	if udpPort != 0 {
		table.Append([]string{"UDP", fmt.Sprintf("0.0.0.0:%d", udpPort)})
	}

	if tcpPort != 0 {
		table.Append([]string{"TCP", fmt.Sprintf("0.0.0.0:%d", tcpPort)})
	}

	fmt.Println(color.GreenString("DNS listening on the following addresses."))
	table.Render()
}
