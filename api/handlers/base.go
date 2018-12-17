package handlers

const (
	StatusOk    = "ok"
	StatusError = "error"
)

type Response struct {
	Status string      `json:"status"`
	Result interface{} `json:"result"`
}
