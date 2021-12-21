package framework

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	restConfig *rest.Config
	kubeClient kubernetes.Interface
	myClient   client.Client
	namespace  string
	name       string
}

func New(
	restConfig *rest.Config,
	kubeClient kubernetes.Interface,
	myClient client.Client,
) *Framework {
	return &Framework{
		restConfig: restConfig,
		kubeClient: kubeClient,
		myClient:   myClient,
		name:       "kfc",
		namespace:  "wow",
	}
}

var (
	RootFramework *Framework
)

func NewInvocation() *Invocation {
	return RootFramework.Invoke()
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework:     f,
		app:           "", //rand.WithUniqSuffix(strings.ToLower(fmt.Sprintf("%s-e2e", DBType))),
		testResources: make([]interface{}, 0),
	}
}

func (i *Invocation) Checker() bool {
	return true
}

type Invocation struct {
	*Framework
	app           string
	testResources []interface{}
}
