FROM registry.svc.ci.openshift.org/openshift/release:golang-1.13
LABEL maintainer="mnairn@redhat.com"

ENV OPERATOR_SDK_VERSION=v0.15.1 \
    OC_VERSION="4.5" \
    GOFLAGS=""

# install oc
RUN curl -Ls https://mirror.openshift.com/pub/openshift-v4/clients/oc/$OC_VERSION/linux/oc.tar.gz | tar -zx && \
    mv oc /usr/local/bin

# install operator-sdk (from git with no history and only the tag)
RUN mkdir -p $GOPATH/src/github.com/operator-framework \
    && cd $GOPATH/src/github.com/operator-framework \
    && git clone --depth 1 -b $OPERATOR_SDK_VERSION https://github.com/operator-framework/operator-sdk \
    && cd operator-sdk \
    && go mod vendor \
    && make tidy \
    && make install \
    && chmod -R 0777 $GOPATH \
    && rm -rf $GOPATH/.cache

COPY delorean /usr/bin/delorean
