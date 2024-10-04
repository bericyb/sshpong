lint: 
	golangci-lint run

server:
	air -c .air_server.toml

client: 
	while true; do \
		go run cmd/client/main.go;\
		echo "Client has exited, restarting...";\
		sleep 2;\
	done

