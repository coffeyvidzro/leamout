package arkesel

type SendRequest struct {
	Sender     string   `json:"sender"`
	Message    string   `json:"message"`
	Recipients []string `json:"recipients"`
}

type MessageData struct {
	Recipient string `json:"recipient"`
	ID        string `json:"id"`
}

type SendResponse struct {
	Status string        `json:"status"`
	Data   []MessageData `json:"data"`
}

type StatusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ID     string `json:"ID"`
		Status string `json:"status"`
	} `json:"data"`
}
