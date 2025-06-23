package shared

type Robot struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	DeviceID string `json:"device_id"`
}
