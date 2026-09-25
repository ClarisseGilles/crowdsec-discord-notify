FROM docker.io/golang:1.27.1-alpine3.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY src/*.go ./
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o /crowdsec-discord-notify .

FROM gcr.io/distroless/static-debian13
COPY --from=build /crowdsec-discord-notify /crowdsec-discord-notify
EXPOSE 8080
ENTRYPOINT ["/crowdsec-discord-notify"]
