module github.com/open-cluster-management/submariner-addon

go 1.16

require (
	github.com/aws/aws-sdk-go v1.38.28
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/mock v1.4.3
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/openshift/api v0.0.0-20210521075222-e273a339932a
	github.com/openshift/build-machinery-go v0.0.0-20210423112049-9415d7ebd33e
	github.com/openshift/library-go v0.0.0-20210609150209-1c980926414c
	github.com/operator-framework/api v0.5.2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/submariner-io/submariner v0.9.0
	github.com/submariner-io/submariner-operator v0.9.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.29.0
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.21.1
	k8s.io/component-base v0.21.1
	k8s.io/klog/v2 v2.8.0
	open-cluster-management.io/addon-framework v0.0.0-20210803032803-58eac513499e
	open-cluster-management.io/api v0.0.0-20210727123024-41c7397e9f2d
	sigs.k8s.io/controller-runtime v0.8.3
)

// ensure compatible between controller-runtime and kube-openapi
replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1

// ensure compatible with submariner-operator
// TODO if submariner has an independent api repo in future, we can remove this
replace k8s.io/client-go v12.0.0+incompatible => k8s.io/client-go v0.21.3
