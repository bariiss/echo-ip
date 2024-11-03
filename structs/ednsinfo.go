package structs

// EDNSInfo struct definition (move to structs package if needed)
type EDNSInfo struct {
	ClientIP    string `json:"client_ip"`
	DNSResolver struct {
		IP       string `json:"ip"`
		Location string `json:"location"`
		Provider string `json:"provider"`
	} `json:"dns_resolver"`
	EDNSInfo struct {
		IP       string `json:"ip"`
		Subnet   string `json:"subnet"`
		Location string `json:"location"`
	} `json:"edns_info"`
}
