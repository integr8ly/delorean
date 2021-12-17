module github.com/integr8ly/delorean

go 1.16

require (
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/aws/aws-sdk-go v1.32.5
	github.com/blang/semver v3.5.1+incompatible
	github.com/bshuster-repo/logrus-logstash-hook v1.0.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v30 v30.1.0
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/integr8ly/cluster-service v0.4.0
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/runc v1.0.0-rc9 // indirect
	github.com/openshift/client-go v0.0.0-20200326155132-2a6cd50aedd0
	github.com/operator-framework/api v0.10.7
	github.com/operator-framework/operator-registry v1.13.6
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/xanzy/go-gitlab v0.31.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
	github.com/openshift/api => github.com/openshift/api v0.0.0-20211028023115-7224b732cc14
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210831095141-e19a065e79f7
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.2.0
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.0.0-beta.5.0.20200123114618-5e3c7d7eb86a

	// Pin to kube 1.22
	k8s.io/api => k8s.io/api v0.0.0-20210716001550-68328c152cca
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20210927122911-41e7589359be
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20210906153829-4a9e16b35712
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20211014130529-c322f47b4613
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20211008010600-d0d3f6d0e754
	k8s.io/client-go => k8s.io/client-go v0.0.0-20210927095136-36ef1697870d
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20210821165119-ef9be83d2fdb
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20210701061254-6e1a91d89121
	k8s.io/code-generator => k8s.io/code-generator v0.22.4-rc.0
	k8s.io/component-base => k8s.io/component-base v0.0.0-20210821161839-63bef0cffea5
	k8s.io/cri-api => k8s.io/cri-api v0.23.0-alpha.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20210816161517-8877a87f724a
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20211014130925-cc33a6ab33de
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20210821165302-7640afa86285
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20210821164430-63daed1faaba
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20210821164756-f980237496c6
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20211008013018-579232b9539e
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20210821164612-e4cb5325257a
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20211014135217-75e926a43c70
	k8s.io/metrics => k8s.io/metrics v0.0.0-20210821163913-98d2fd1dc73d
	k8s.io/node-api => k8s.io/node-api v0.0.0-20200730165709-03155dcb9a22
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20210821163016-69619bf0ea77
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20210821164248-1e455820cbc7
	k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20210821163350-065a92b2a5e1
)
