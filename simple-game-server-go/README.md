# Simple Game Server (Go)
A very simple game server designed to demonstrate and test running a game server on the [Unity Multiplay platform](https://unity.com/products/multiplay).

The [prebuilt releases](https://github.com/Unity-Technologies/multiplay-examples/releases) are ready for you to upload to Multiplay to try out the service without writing a single line of code!

The capabilities of this sample are as follows:

- Handling of the [Multiplay allocation lifecycle](https://docs.unity.com/game-server-hosting/en/manual/concepts/allocation-lifecycle)
    - When allocated, the sample starts a TCP server on the port defined in the `server.json` file
    - When de-allocated, this TCP server is stopped
- Dynamic server query results
    - Data such as number of players, map name, etc. are handled appropriately
    - `sqp` and `a2s` query protocols supported over the query port defined in the `server.json` file
- Backfill allocation keep alive
    - If `"enableBackfill"="true"` is set on the `server.json` then the server will support keeping alive a backfill ticket in the matchmaker
    - If you are not using the production matchmaker gateway URL (`https://matchmaker.services.api.unity.com`), then you can change this location by setting the `matchmakerUrl` parameter in your [build configuration settings](https://docs.unity.com/game-server-hosting/en/manual/guides/manage-build-configurations) 
        - e.g. `"matchmakerUrl": "https://matchmaker-stg.services.api.unity.com"`
    - Please see the [Matchmaker docs on configuring backfill](https://unity-technologies.github.io/ucg-matchmaking-docs/standard/backfill-tutorial) for more information on backfill
- Client simulation
    - A game client can be simulated by opening a TCP connection once the server is allocated
        - The server should be allocated first by invoking the [server allocations API](https://services.docs.unity.com/multiplay-gameserver/v1/index.html#tag/Allocations/operation/ProcessAllocation)
        - A TCP connection can be opened with a `netcat` command: `nc <ip> <port>`. Once the connection is opened, this connection represents a player - as such, the 'Concurrent Users' count will be updated in the dashboard
