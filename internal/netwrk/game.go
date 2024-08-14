package netwrk

import (
	"log"
	"net"
)

func handleGameConnection(conn net.Conn) {
	defer conn.Close()

	messageBytes := make([]byte, 126)

	n, err := conn.Read(messageBytes)
	if err != nil {
		log.Printf("Error reading game ID on connection", err)
	}

	gameID := string(messageBytes[:n])
	if err != nil {
		log.Printf("Game id was not a string?", err)
	}

	clientChan := make(chan GameMessage)

	n, err = conn.Read(messageBytes)
	if err != nil {
		log.Printf("Error reading message %v", err)
		return
	}

	gameClients, ok := gameChans.games[gameID]
	if !ok {
		newGameClients := GameClients{
			client1: clientChan,
			client2: nil,
		}
		gameChans.games[gameID] = newGameClients
	} else {
		gameClients.client2 = clientChan
	}
}

func handleGameMessage(conn net.Conn, message *GameMessage) error {
	return nil
}
