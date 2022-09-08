FROM golang:1.19-alpine as build

RUN apk update && apk add --no-cache gcc musl-dev git

WORKDIR /app

COPY . .

RUN go build -ldflags '-w -s' -a -o ./bin/k2h *.go

FROM alpine

RUN apk update && apk add --no-cache bash
COPY --from=build /app/bin/k2h /opt/

CMD ["/opt/k2h"]
