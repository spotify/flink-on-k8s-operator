module github.com/spotify/flink-on-k8s-operator

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/hashicorp/go-version v1.3.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	golang.org/x/net v0.0.0-20210716203947-853a461950ff
	golang.org/x/tools v0.1.5 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.9.3
	volcano.sh/apis v0.0.0-20210518032656-21e2239e42bc
)
