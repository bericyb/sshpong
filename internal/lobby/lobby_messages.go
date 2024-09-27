package lobby

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"strings"
)

type LobbyMessage struct {
	MessageType string `json:"message_type"`
	Message     any    `json:"message"`
}

type Name struct {
	Name string `json:"name"`
}

type Chat struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

type Invite struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type PendingInvite struct {
	Recipient string `json:"recipient"`
}

type Accept struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Accepted struct {
	Accepter string `json:"accepter"`
	GameID   string `json:"game_id"`
}

type StartGame struct {
	To     string `json:"to"`
	GameID string `json:"game_id"`
}

type Decline struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Disconnect struct {
	From string `json:"from"`
}

type Connect struct {
	From string `json:"from"`
}

type Error struct {
	Message string `json:"message"`
}

func Marshal(a LobbyMessage) ([]byte, error) {
	slog.Debug("Marshalling message", slog.Any("message type", a.MessageType))
	bm, err := json.Marshal(a.Message)
	if err != nil {
		return nil, err
	}

	a.Message = bm
	return json.Marshal(a)
}

// Use this to get the appropriate message type into the message field then assert the
// right struct accordingly to safely access the fields you need.
func Unmarshal(b []byte) (LobbyMessage, error) {
	lm := LobbyMessage{}
	err := json.Unmarshal(b, &lm)
	if err != nil {
		return lm, err
	}

	smsg, ok := lm.Message.(string)
	if !ok {
		slog.Debug("error asserting message to string")
	}
	slog.Debug("type of message", slog.Any("type of message", reflect.TypeOf(smsg)), slog.String("message", smsg))

	jsonBytes, err := base64.StdEncoding.DecodeString(smsg)
	lm.Message = jsonBytes

	switch strings.ToLower(lm.MessageType) {
	case "name":
		n := Name{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}

		err := json.Unmarshal(bs, &n)
		if err != nil {
			slog.Debug("Error", slog.Any("error", err))
			return lm, err
		}
		lm.Message = n
		return lm, nil
	case "chat":
		c := Chat{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &c)
		if err != nil {
			slog.Debug("chat", slog.Any("error", err))
			return lm, err
		}
		lm.Message = c
		return lm, nil
	case "invite":
		i := Invite{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &i)
		if err != nil {
			slog.Debug("invite", slog.Any("error", err))
			return lm, err
		}
		lm.Message = i
		return lm, nil
	case "pending_invite":
		pi := PendingInvite{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &pi)
		if err != nil {
			slog.Debug("pending_invite", slog.Any("error", err))
			return lm, err
		}
		lm.Message = pi
	case "accept":
		a := Accept{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &a)
		if err != nil {
			slog.Debug("accept", slog.Any("error", err))
			return lm, err
		}
		lm.Message = a
		return lm, nil
	case "accepted":
		a := Accepted{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &a)
		if err != nil {
			slog.Debug("accepted", slog.Any("error", err))
			return lm, err
		}
		lm.Message = a
		return lm, nil
	case "start_game":
		sg := StartGame{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &sg)
		if err != nil {
			slog.Debug("start_game", slog.Any("error", err))
			return lm, err
		}
		lm.Message = sg
		return lm, nil
	case "decline":
		d := Decline{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &d)
		if err != nil {
			slog.Debug("decline", slog.Any("error", err))
			return lm, err
		}
		lm.Message = d
		return lm, nil
	case "disconnect":
		di := Disconnect{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &di)
		if err != nil {
			slog.Debug("disconnect", slog.Any("error", err))
			return lm, err
		}
		lm.Message = di
		return lm, nil
	case "connect":
		co := Connect{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &co)
		if err != nil {
			slog.Debug("connect", slog.Any("error", err))
			return lm, err
		}
		lm.Message = co
		return lm, nil
	case "error":
		e := Error{}
		bs, ok := lm.Message.([]byte)
		if !ok {
			return lm, err
		}
		err := json.Unmarshal(bs, &e)
		if err != nil {
			slog.Debug("error", slog.Any("error", err))
			return lm, err
		}
		lm.Message = e
		return lm, nil
	default:
		return lm, errors.New("unknown message type")
	}
	return lm, errors.New("unknown message type")
}
