FROM golang:1.26.1-alpine AS build
WORKDIR /src
COPY test/integration/e2e/telegrammock/api.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/telegram-mock ./api.go

FROM alpine:3.20
RUN adduser -D appuser
USER appuser
COPY --from=build /out/telegram-mock /usr/local/bin/telegram-mock
ENTRYPOINT ["telegram-mock"]

