package schema

const (
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	MongoImage                  string = "mongo"
	JobName                     string = "demo-job"
)

// Extra stuffs

const (
	VolumeNameForPod string = "random-volume-1"
	InitScriptName   string = "init.js"
	KeyNameForVolume string = "examplefile"
	KeyPathForVolume string = "mypath"
)
const (
	// Roles
	MongoReaderWriterRole string = "mongodb-reader-writer-role"
	MongoReaderRole       string = "mongodb-reader-role"
	MongoDBAdminRole      string = "mongodb-db-admin-role"

	// Secret Access requests
	MongoDBReadWriteSecretAccessRequest string = "mongodb-read-write-access-req"
	MongoDBReadSecretAccessRequest string = "mongodb-read-access-req"
	MongoDBAdminSecretAccessRequest 	string = "mongodb-admin-access-req"

	// Service Accounts
	MongoDBReadWriteServiceAccount string = "mongodb-read-write-sa"
	MongoDBReadServiceAccount string = "mongodb-read-sa"
	MongoDBAdminServiceAccount 	   string = "mongodb-admin-sa"
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
func makeImageNameFromVersion(version string) string {
	return MongoImage + ":" + version
}



// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
