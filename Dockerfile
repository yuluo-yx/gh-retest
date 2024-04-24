FROM ubuntu:latest

RUN apt-get update && apt-get install -y curl

RUN curl -O https://dl.google.com/go/go1.22.2.linux-amd64.tar.gz
RUN tar -xvf go1.22.2.linux-amd64.tar.gz
RUN mv -f go /usr/local

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="/go/bin:${PATH}"

RUN go version

LABEL "com.github.actions.name"="gh-retest"
LABEL "com.github.actions.description"="gh-retest"
LABEL "com.github.actions.icon"="home"
LABEL "com.github.actions.color"="red"

LABEL "repository"="https://github.com/yuluo-yx/gh-retest"
LABEL "homepage"="https://github.com/yuluo-yx/gh-retest"
LABEL "maintainer"="yuluo <yuluo08290126@gmail.com>"

LABEL "Name"="Github Pull Request Retest"

WORKDIR /app

COPY retest/ retest/
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY LICENSE LICENSE
COPY README.md README.md
COPY action.yml action.yml

RUN go build -o /usr/local/bin/retest main.go && \
    chmod +x /usr/local/bin/retest && \
    ls -l /usr/local/bin/retest

CMD ["retest"]
