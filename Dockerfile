FROM golang:1.20.1 as build
WORKDIR /go/src/github.com/natrontech/alertmanager-uptime-kuma-push
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o alertmanager-uptime-kuma-push ./cmd/pusher

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=build /go/src/github.com/natrontech/alertmanager-uptime-kuma-push/alertmanager-uptime-kuma-push ./
EXPOSE 8081
CMD ["./alertmanager-uptime-kuma-push"]
