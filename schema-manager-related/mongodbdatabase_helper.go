package schema

const (
	MongoDBDatabasePlugin       string = "mongodb-database-plugin"
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	ResourceKindMongoDBRole     string = "MongoDBRole"
	UserNamespace               string = "dev"
	MongoImage                  string = "mongo"
	JobName                     string = "demo-job"
	MongoDBDatabaseSchemaName   string = "mydb"
)

func checkPrefixMatch(secretName, accessRequestName string) bool {
	if len(accessRequestName) > len(secretName) {
		return false
	}
	for i := 0; i < len(accessRequestName); i++ {
		if accessRequestName[i] != secretName[i] {
			return false
		}
	}
	return true
}

func makeSecretEngineName(name string) string {
	return name + "-secret-engine"
}

const (
	// Roles
	MongoReaderWriterRole string = "mongodb-reader-writer-role"
	MongoReaderRole       string = "mongodb-reader-role"
	MongoDBAdminRole      string = "mongodb-db-admin"

	// Secret Access requests
	MongoDBReadWriteSecretAccessRequest string = "mongodb-read-write-access-req"

	// Service Accounts
	MongoDbReadWriteServiceAccount string = "mongodb-read-write-sa"
)
