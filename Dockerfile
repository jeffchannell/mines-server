FROM golang:1.12.7-alpine3.10 AS build
RUN apk add git
WORKDIR /go/src/app
COPY . .
RUN go get .../
RUN GOOS=linux go build -ldflags="-s -w" -o ./bin/mines-server ./main.go

FROM alpine:3.10
RUN apk --no-cache add ca-certificates
WORKDIR /usr/bin
COPY --from=build /go/src/app/bin /go/bin
EXPOSE 8080
ENTRYPOINT /go/bin/mines-server