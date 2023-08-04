FROM unitymultiplay/linux-base-image:latest

USER root
COPY ./simple-game-server-go /usr/local/bin/
USER mpukgame

ENTRYPOINT ["simple-game-server-go"]
