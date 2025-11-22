# syntax=docker/dockerfile:1

FROM golang:1.19

WORKDIR /app

COPY ./bin/linux_amd64/api .

EXPOSE 8080

CMD ["/api"]
