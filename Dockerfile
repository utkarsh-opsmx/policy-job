FROM golang:1.22 AS builder

WORKDIR /go/src/github.com/OpsMx/argocd-policy-plugin/

COPY go.* ./
RUN go mod download

COPY . .
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin

RUN GOOS="linux" GOARCH="amd64" CGO_ENABLED=0 go build -o policy-job *.go

########################################
# Final policy-job stage
########################################

FROM alpine:3.18.4 AS policy-job

COPY --from=builder /go/src/github.com/OpsMx/argocd-policy-plugin/policy-job /usr/local/bin/policy-job
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/kubectl

RUN apk update