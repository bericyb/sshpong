package netwrk

import (
	"log"
	"net"
	sync "sync"

	"google.golang.org/protobuf/proto"
)

type Client struct {
	Username string
	Conn     net.Conn
}

type LobbyPlayersMessage struct {
	Type        string
	Username    string
	IsAvailable chan bool
}

type ExternalMessage struct {
	Target  string
	Message *LobbyMessage
}

var lobbyListener chan LobbyPlayersMessage
var externalMessageChan chan ExternalMessage

var lobbyMembers sync.Map

func init() {
	lobbyListener = make(chan LobbyPlayersMessage)
	externalMessageChan = make(chan ExternalMessage)

	lobbyMembers = sync.Map{}

	go func() {
		for {
			msg := <-externalMessageChan
			player, ok := lobbyMembers.Load(msg.Target)
			if !ok {
				log.Println("failed to send to target", msg.Target)
				continue
			}
			client, _ := player.(Client)
			bytes, _ := proto.Marshal(msg.Message)
			_, err := client.Conn.Write(bytes)
			if err != nil {
				log.Println("Could not write to target", msg.Target, err)
			}

		}
	}()
}

// Starts listening on port 12345 for TCP connections
// Also creates client pool and game connection singletons
func LobbyListen() {

	listener, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleLobbyConnection(conn)
	}
}

func GamesListen() {

	gameListener, err := net.Listen("tcp", "127.0.0.1:42069")
	if err != nil {
		log.Fatal(err)
	}

	defer gameListener.Close()
	for {
		conn, err := gameListener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		handleGameConnection(conn)
	}
}
