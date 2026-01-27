package acpclient

type AcpConfig struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}
