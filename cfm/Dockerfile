FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git
ENV CGO_ENABLED=0

WORKDIR /cfmid

COPY go.mod ./
RUN go mod download

COPY . .
RUN go get github.com/xuri/excelize/v2 && go mod tidy && go build -o cfm-server wrapper.go

FROM wishartlab/cfmid:latest
WORKDIR /cfmid

COPY --from=builder /cfmid/cfm-server .

RUN chmod +x cfm-server

EXPOSE 5001
ENTRYPOINT ["./cfm-server"]
