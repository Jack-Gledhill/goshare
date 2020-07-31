FROM golang:latest

COPY . /opt/site

WORKDIR /opt/site

RUN go get github.com/BadgeBot/gogger
RUN go build -o main .

CMD ["/opt/site/main"]