package framework

import (
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
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

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework:     f,
		app:           "", //rand.WithUniqSuffix(strings.ToLower(fmt.Sprintf("%s-e2e", DBType))),
		testResources: make([]interface{}, 0),
	}
}

type Invocation struct {
	*Framework
	app           string
	testResources []interface{}
}

func NewInvocation() *Invocation {
	return RootFramework.Invoke()
}

func (i *Invocation) Checker() bool {
	return true
}

type DBOptions struct {
	DBType         string
	SslModeEnabled bool
}
type SchemaOptions struct {
	AutoApproval bool
	ToRestore    bool
}

type TestOptions struct {
	*Invocation
	Mongodb        *kdm.MongoDB
	Vault          *kvm_server.VaultServer
	SchemaDatabase *schemav1alpha1.MongoDBDatabase
	InitJob        *batchv1.Job
	RestoreSession *stash.RestoreSession
	*DBOptions
	*SchemaOptions
	Secret *core.Secret
}
