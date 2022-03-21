# Simple Game Server (Go)
A very simple game server designed to demonstrate and test running a game server on the [Unity Multiplay platform](https://unity.com/products/multiplay).

The [prebuilt releases](https://github.com/Unity-Technologies/multiplay-examples/releases) are ready for you to upload to Multiplay to try out the service without writing a single line of code!

The capabilities of this sample are as follows:

- Handling of the [Multiplay allocation lifecycle](https://docs.unity.com/multiplay/Content/shared/allocation-flow.htm)
    - Achieved by watching for file events on the provided configuration file
    - When allocated, the sample starts a TCP server on the configured `-port` flag which listens for client connections
    - When de-allocated, this TCP server is stopped
- Dynamic server query results
    - Data such as number of players, map name, etc. are handled appropriately
    - `sqp` and `a2s` query protocols over the configured UDP `-queryport` flag
- Backfill allocation keep alive
    - If `"enableBackfill"="true"` is set on the `server.json` then the server will support keeping alive a backfill ticket in the matchmaker
    - If you are not using the production matchmaker gateway url (`https://matchmaker.services.api.unity.com`), then you can change this location by setting the `matchmakerUrl` param in `server.json`. 
        - e.g. `"matchmakerUrl": "https://matchmaker-stg.services.api.unity.com"`
    - Please see the [Matchmaker docs on configuring backfill](https://unity-technologies.github.io/ucg-matchmaking-docs/standard/backfill-tutorial) for more information on backfill.