FROM docker.io/golang:1.22.2 AS builder

WORKDIR /workspace

COPY retest/ retest/
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY README.md README.md
COPY LICENSE LICENSE
COPY action.yml action.yml

RUN CGO_ENABLED=0 go build -o gh-retest .

FROM docker.io/library/ubuntu:23.10

COPY --from=builder /workspace/gh-retest /usr/local/bin/gh-retest
COPY --from=builder /workspace/LICENSE /LICENSE
COPY --from=builder /workspace/README.md /README.md

LABEL "com.github.actions.name"="gh-retest"
LABEL "com.github.actions.description"="gh-retest"
LABEL "com.github.actions.icon"="home"
LABEL "com.github.actions.color"="red"

LABEL "repository"="https://github.com/yuluo-yx/gh-retest"
LABEL "homepage"="https://github.com/yuluo-yx/gh-retest"
LABEL "maintainer"="yuluo <yuluo08290126@gmail.com>"

LABEL "Name"="Github Pul Request Retest"

RUN apt update -y && \
    apt install -y curl

CMD ["gh-retest"]
