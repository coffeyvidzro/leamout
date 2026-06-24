package provider

type Message struct {
	To      string
	Content string
	From    string
}

type Result struct {
	MessageID string
	Status    string
	Provider  string
}
