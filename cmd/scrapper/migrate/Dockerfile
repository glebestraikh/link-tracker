FROM golang:1.26.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/scrapper-migrate ./cmd/scrapper/migrate

FROM alpine:3.20
RUN adduser -D appuser
USER appuser
COPY --from=build /out/scrapper-migrate /usr/local/bin/scrapper-migrate
ENTRYPOINT ["scrapper-migrate"]