package netwrk

import (
	"fmt"
	"net"

	"google.golang.org/protobuf/proto"
)

func ConnectToLobby(username string) (net.Conn, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		return nil, fmt.Errorf("Sorry, failed to connect to server...")
	}

	loginMsg, err := proto.Marshal(&LobbyMessage{Type: "name", Content: username})
	if err != nil {
		return nil, fmt.Errorf("Sorry bro but your username is wack AF...")
	}

	_, err = conn.Write(loginMsg)
	if err != nil {
		return nil, fmt.Errorf("Sorry, could not communicate with server...")
	}

	return conn, nil
}
