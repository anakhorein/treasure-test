# Compile stage
FROM golang:1.21 AS build-env

ADD . /dockerdev
WORKDIR /dockerdev

RUN CGO_ENABLED=0 go build -o /server

# Final stage
FROM centos:latest

EXPOSE 8080

WORKDIR /
COPY --from=build-env /server /

CMD ["/server"]