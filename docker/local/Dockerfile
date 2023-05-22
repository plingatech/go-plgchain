FROM golang:1.18-alpine AS builder

WORKDIR /go-plgchain

ADD go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o go-plgchain main.go

FROM alpine:latest AS runner

RUN apk --no-cache add ca-certificates jq

WORKDIR /go-plgchain

COPY --from=builder /go-plgchain/go-plgchain ./
COPY ./docker/local/go-plgchain.sh ./
COPY ./core-contracts/artifacts ./core-contracts/artifacts

# Expose json-rpc, libp2p and grpc ports
EXPOSE 8545 9632 1478 5001

ENTRYPOINT ["./go-plgchain.sh"]
