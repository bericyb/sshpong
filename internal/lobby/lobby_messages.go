package lobby

import "encoding/json"

const (
	Name = iota
	Chat
	Invite
	PendingInvite
	Accept
	Accepted
	StartGame
	Decline
	Disconnect
	Connect
	CurrentlyConnected
	Error
)

type NameData struct {
	Name string `json:"name"`
}

type ChatData struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

type InviteData struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type PendingInviteData struct {
	Recipient string `json:"recipient"`
}

type AcceptData struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type AcceptedData struct {
	Accepter string `json:"accepter"`
	GameID   string `json:"game_id"`
}

type StartGameData struct {
	To     string `json:"to"`
	GameID string `json:"game_id"`
}

type DeclineData struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type DisconnectData struct {
	From string `json:"from"`
}

type ConnectData struct {
	From string `json:"from"`
}

type CurrentlyConnectedData struct {
	Players []string `json:"players"`
}

type ErrorData struct {
	Message string `json:"message"`
}

func Unmarshal[T NameData | ChatData | InviteData | PendingInviteData | AcceptData | AcceptedData | StartGameData | DeclineData | DisconnectData | ConnectData | CurrentlyConnectedData | ErrorData](msg []byte) (T, error) {
	var d T
	err := json.Unmarshal(msg[1:], &d)
	if err != nil {
		return d, err
	}
	return d, nil
}

func Marshal[T NameData | ChatData | InviteData | PendingInviteData | AcceptData | AcceptedData | StartGameData | DeclineData | DisconnectData | ConnectData | CurrentlyConnectedData | ErrorData](msg T, header int) ([]byte, error) {
	mb, err := json.Marshal(msg)
	if err != nil {
		return mb, err
	}

	b := []byte{byte(header)}

	b = append(b, mb...)

	return b, nil
}
