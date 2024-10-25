
# SSH Pong

SSHPong is a multiplayer Pong game designed for a command-line interface, allowing players to connect and play through SSH. The project includes a central server to manage game state and clients that communicate with it for real-time gameplay.

## Features

- Multiplayer Pong over SSH
- Central server for game state synchronization
- Real-time updates between clients

## Getting Started

### Prerequisites

- [Go](https://golang.org/) (for compiling and running the server and client code)
- SSH access (for multiplayer interaction)

### Installation

1. **Clone the Repository**

    ```bash
    git clone https://github.com/yourusername/sshpong.git
    cd sshpong
    ```

2. **Build the Server and Client**

    ```bash
    go build -o server ./cmd/server/main.go
    go build -o client /cmd/client/main.go
    ```

### Running the Game

1. **Start the Server**

    ```bash
    ./server
    ```

2. **Connect as a Player**

    Run the client program to join the game:

    ```bash
    ./client
    ```

### Controls

- Use the W and S keys to move your paddle up and down.
- Press `q` to quit the game.

## Contributing

Contributions are welcome! Please submit a pull request or open an issue to discuss improvements.

## License

This project is licensed under the MIT License.
