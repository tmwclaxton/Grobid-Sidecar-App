FROM golang:1.20

WORKDIR /go/app

ADD . .

RUN useradd -ms /bin/bash development
USER development
RUN go get .

CMD ["go", "run", "main.go"]
#CMD ["dlv", "--headless", "--listen=:40000", "--api-version=2", "exec", "main.go"]
#CMD ["dlv", "debug", "--headless", "--listen=:40000", "--api-version=2", "--accept-multiclient", "exec", "main.go"]