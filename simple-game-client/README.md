# Simple Game Client

This is a very simple game client, designed to be used with the simple-matchmaker.

This app is designed to provide the simplest complete example of using the Multiplay. It uses a very simple matchmaker
which is designed to demonstrate flows that need to be made and is not designed for production use.

## Expected flow:

- Simple-game-client app starts
- Creates a Player UUID unique for the game client run.
- Repeatedly call the simple-matchmaker `/player` endpoint with the player UUID
- Eventually the endpoint will return an IP and Port to connecto
- Simple-game-client app connects to port using a basic TCP connection
- App periodically sends messages and displays anything it receives from the connection.
