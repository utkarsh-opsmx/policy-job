#!/usr/bin/env bash

case $(uname -m) in
	x86_64)
		ARCH=amd64;;
	aarch64)
		ARCH=arm64;;
	*)
		echo Unknown architecture $(uname -m)
		exit 1;;
esac

echo Using architecture ${ARCH}

DEST=/usr/local/bin

# Install Helm
curl -SL https://get.helm.sh/helm-${HELM_VERSION}-linux-${ARCH}.tar.gz | tar -xz linux-${ARCH}/helm && mv linux-${ARCH}/helm ${DEST}

# Install Kustomize
curl -SL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$(echo ${KUSTOMIZE_VERSION}|tr -d kustomize/)/kustomize_$(echo ${KUSTOMIZE_VERSION}|tr -d kustomize/)_linux_${ARCH}.tar.gz | tar -xzC ${DEST}