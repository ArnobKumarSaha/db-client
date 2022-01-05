package framework

const (
	MongoDBDatabaseSchemaName  = "sample"
	MongoDBVersion             = "4.4.6"
	MongoDBName                = "mng-shrd"
	VaultName                  = "vault"
	SchemaName                 = "mydb"
	CollectionName             = "databases"
	SubjectName                = "sa_name"
	RepositoryName             = "local-repo"
	BackupConfigName           = "mongo-backup"
	ConfigMapName              = "test-cm"
	SecretName                 = "test-secret"
	LinodeRepositorySecretName = "linode-secret-stash"
	MinioRepositorySecretName  = "minio-secret"
	RunnerJobName              = "kubernetes-go-test"
	ServiceAccountName         = "my-test-sa"
	OperatorImageName          = "arnobkumarsaha/kubernetes-go-test"
)

const (
	CommonNamespace   = "dev"
	SchemaNamespace   = "dev"
	VaultNamespace    = "demo"
	DatabaseNamespace = "db"
)

const (
	StandAlone string = "standalone"
	ReplicaSet string = "replicaset"
	Sharded    string = "sharded"
)
