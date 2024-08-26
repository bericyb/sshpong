package netwrk

import (
	"log"
	"net"
	"sshpong/internal/pong"
	"strings"
	sync "sync"

	"google.golang.org/protobuf/proto"
)

type Client struct {
	Username string
	Conn     net.Conn
}

type ExternalMessage struct {
	Target  string
	Message *LobbyMessage
}

type GameClients struct {
	Client1 Client
	Client2 Client
}

var externalMessageChan chan ExternalMessage

var lobbyMembers sync.Map
var games sync.Map

func init() {
	externalMessageChan = make(chan ExternalMessage)

	lobbyMembers = sync.Map{}
	games = sync.Map{}

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

		go func(conn net.Conn) {
			messageBytes := make([]byte, 126)

			n, err := conn.Read(messageBytes)
			if err != nil {
				log.Printf("Error reading game ID on connection %s", err)
			}

			gInfo := strings.SplitAfter(string(messageBytes[:n]), ":")
			if err != nil {
				log.Printf("Game id was not a string? %s", err)
			}

			game, ok := games.Load(gInfo[0])
			if !ok {
				games.Store(gInfo[0], GameClients{Client1: Client{
					Username: gInfo[1],
					Conn:     conn,
				}, Client2: Client{}})
			} else {
				gameclients, _ := game.(GameClients)
				client2 := Client{
					Username: gInfo[1],
					Conn:     conn,
				}

				games.Store(gInfo[0], GameClients{
					Client1: gameclients.Client1,
					Client2: client2})

				go pong.StartGame(gameclients.Client1.Conn, client2.Conn, gameclients.Client1.Username, client2.Username)
			}
		}(conn)
	}
}
