package schema

const (
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	MongoImage                  string = "mongo"
	JobName                     string = "demo-job"
)

// Extra stuffs

const (
	VolumeNameForPod string = "random-volume-1"
	VolumeMountPath  string = "/etc/config"
	InitScriptName   string = "init.js"
	KeyNameForVolume string = "examplefile"
	KeyPathForVolume string = "mypath"
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
