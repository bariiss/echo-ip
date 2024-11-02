package structs

type GeoInfo struct {
	ClientIP  string  `json:"client_ip"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Timezone  string  `json:"timezone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	ISP       string  `json:"isp"`
}
