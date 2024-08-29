FROM registry.access.redhat.com/ubi9/ubi:latest
LABEL maintainer="mnairn@redhat.com"

ENV OPERATOR_SDK_VERSION=v1.21.0 \
    GOFLAGS="" \
    PATH=$PATH:/usr/local/go/bin

COPY resources/2015-IT-Root-CA.pem resources/2022-IT-Root-CA.pem /etc/pki/ca-trust/source/anchors/

RUN set -o pipefail && \
    INSTALL_PKGS="make rsync wget gcc which git diffutils openssh" && \
    yum install -y --setopt=tsflags=nodocs $INSTALL_PKGS && \
    curl -Ls https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.15.28/openshift-client-linux.tar.gz | tar -zx && \
    mv oc /usr/local/bin && \
    curl -Ls https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.15.28/opm-linux.tar.gz | tar -zx && \
    mv opm /usr/local/bin && \
    curl -Lo /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/v4.35.2/yq_linux_amd64 && \
    chmod +x /usr/local/bin/yq && \
    curl -Lo /usr/local/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.7/jq-linux64 && \
    chmod +x /usr/local/bin/jq && \
    wget https://golang.org/dl/go1.20.linux-amd64.tar.gz && \
    tar -zxvf go1.20.linux-amd64.tar.gz -C /usr/local/ && \
    curl -Ls https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v4.5.7/kustomize_v4.5.7_linux_amd64.tar.gz | tar -zx && \
    mv kustomize /usr/local/bin && \
    chmod +x /usr/local/bin/kustomize

# install operator-sdk
RUN curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_linux_amd64 && \
    mv operator-sdk_linux_amd64 /usr/bin/operator-sdk && \
    chmod +x /usr/bin/operator-sdk

COPY scripts/prow /usr/local/bin
COPY scripts/delorean /usr/local/bin
COPY scripts/ocm /usr/local/bin
COPY scripts/jenkins-reports /usr/local/bin
COPY build/delorean /usr/local/bin/delorean
COPY configurations /etc/delorean/configurations
RUN chmod -R +r /etc/delorean/configurations
RUN mkdir -p /.ssh
RUN update-ca-trust
