FROM centos:7
LABEL maintainer="mnairn@redhat.com"

ENV OPERATOR_SDK_VERSION=v0.15.1 \
    OC_VERSION="4.5" \
    GOFLAGS=""

# install oc
RUN curl -Ls https://mirror.openshift.com/pub/openshift-v4/clients/oc/$OC_VERSION/linux/oc.tar.gz | tar -zx && \
    mv oc /usr/local/bin

# install operator-sdk
RUN curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu && \
    mv operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu /usr/bin/operator-sdk && \
    chmod +x /usr/bin/operator-sdk

COPY build/delorean /usr/bin/delorean
