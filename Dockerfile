FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY . ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -a -installsuffix cgo -o templ-demo .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/templ-demo .

CMD ["./templ-demo"]
