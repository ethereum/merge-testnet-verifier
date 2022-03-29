FROM golang:1.17.8-alpine

RUN apk add --update gcc musl-dev

WORKDIR /app

COPY . .

RUN go build .

ENTRYPOINT ["./merge_testnet_verifier"]