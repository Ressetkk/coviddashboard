FROM golang:alpine AS build

WORKDIR /src
COPY . .
RUN go build -o /build/agent agent/main.go

FROM alpine:latest
COPY --from=build /build/agent /agent
CMD "/agent"