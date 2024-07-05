FROM golang:1.19-bullseye

ARG VERSION

# Install deps
WORKDIR /app
COPY go.mod go.sum /app/
RUN go mod download

COPY . /app

RUN go build -o ./bin/streamer ./streamer

ENV GAE_VERSION=$VERSION

EXPOSE 6000
USER nobody
ENTRYPOINT ["./bin/streamer"]