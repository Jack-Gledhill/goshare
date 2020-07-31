FROM golang:latest

COPY . /opt/site

WORKDIR /opt/site

RUN go build -o main .

CMD ["/opt/site/main"]