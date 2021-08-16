package types

//RequestResponse object contains the request belongs to reverse-proxy
type RequestResponse struct {
	Body    []byte              `json:"body"`
	Headers map[string][]string `json:"headers"`
}
