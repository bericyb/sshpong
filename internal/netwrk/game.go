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

	_ = string(messageBytes[:n])
	if err != nil {
		log.Printf("Game id was not a string?", err)
	}

	_ = make(chan GameMessage)

	n, err = conn.Read(messageBytes)
	if err != nil {
		log.Printf("Error reading message %v", err)
		return
	}

}

func handleGameMessage(conn net.Conn, message *GameMessage) error {
	return nil
}
