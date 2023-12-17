FROM golang:1.14.2

WORKDIR /go/app

ADD . .

RUN useradd -ms /bin/bash development
USER development

RUN go get .

CMD ["go", "run", "main.go"]
