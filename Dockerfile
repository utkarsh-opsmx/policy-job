FROM golang:1.24 AS builder

ARG KUSTOMIZE_VERSION=v5.0.3
ARG HELM_VERSION=v3.13.2
ARG GIT_VERSION=2.48.1

WORKDIR /go/src/github.com/OpsMx/argocd-policy-plugin/

COPY go.* ./
RUN go mod download

COPY . .
RUN ./deps.sh

RUN GOOS="linux" GOARCH="amd64" CGO_ENABLED=0 go build -o policy-job *.go

########################################
# Final policy-job stage
########################################

FROM alpine:3.18.4 AS policy-job

COPY --from=builder /go/src/github.com/OpsMx/argocd-policy-plugin/policy-job /usr/local/bin/policy-job

RUN apk update