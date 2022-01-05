package framework

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	restConfig        *rest.Config
	kubeClient        kubernetes.Interface
	myClient          client.Client
	schemaNamespace   string
	vaultNamespace    string
	databaseNamespace string
}

func New(
	restConfig *rest.Config,
	kubeClient kubernetes.Interface,
	myClient client.Client,
) *Framework {
	return &Framework{
		restConfig:        restConfig,
		kubeClient:        kubeClient,
		myClient:          myClient,
		schemaNamespace:   SchemaNamespace,
		vaultNamespace:    VaultNamespace,
		databaseNamespace: DatabaseNamespace,
	}
}

func (f *Framework) SetNamespace(db, vault, schema string) {
	f.schemaNamespace = schema
	f.vaultNamespace = vault
	f.databaseNamespace = db
}

func (f *Framework) SetSameNamespace() {
	f.schemaNamespace = CommonNamespace
	f.vaultNamespace = CommonNamespace
	f.databaseNamespace = CommonNamespace
}

var (
	RootFramework *Framework
)

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework:     f,
		app:           "",
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
	AutoApproval       bool
	ToRestore          bool
	SchemaDatabaseName string
	VolumeSourceSecret bool
}

type VaultOptions struct {
}

type TestOptions struct {
	*Invocation
	Mongodb        *kdm.MongoDB
	Vault          *kvm_server.VaultServer
	SchemaDatabase *schemav1alpha1.MongoDBDatabase
	RunnerJob      *batchv1.Job
	*DBOptions
	*SchemaOptions
	*VaultOptions
}
