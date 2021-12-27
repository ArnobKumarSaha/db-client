package framework

const (
	MongoDBDatabaseSchemaName = "sample"
	MongoDBName               = "mng-shrd"
	VaultName                 = "vault"
	SchemaName                = "mydb"
	SubjectName               = "sa_name"
	RepositoryName            = "local-repo"
	ConfigMapName             = "test-cm"
	RepositorySecretName      = "linode-secret-stash"
	RunnerJobName             = "kubernetes-go-test"
	ServiceAccountName        = "my-test-sa"
)

const (
	StandAlone string = "standalone"
	ReplicaSet string = "replicaset"
	Sharded    string = "sharded"
)
