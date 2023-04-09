FROM golang:1.19-alpine3.17 AS build

WORKDIR /app/

COPY . .

RUN go mod download
RUN go build -o studybot

FROM alpine:latest

WORKDIR /app/

COPY --from=build /app/studybot .
COPY --from=build /app/env /env

CMD ["./studybot"]