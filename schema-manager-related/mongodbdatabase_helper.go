package schema

const (
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	MongoImage                  string = "mongo"
	JobName                     string = "demo-job"
)

// Extra stuffs

const (
	VolumeNameForPod    string = "random-volume-1"
	InitScriptName      string = "init.js"
	KeyNameForVolume    string = "examplefile"
	KeyPathForVolume    string = "mypath"
	MongoInitScriptPath string = "/init-scripts"
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

const MongoPrefix = "-mongo"

// For Admin
func getMongoAdminRoleName(dbName string) string {
	return dbName + MongoPrefix + "-admin-role"
}
func getMongoAdminSecretAccessRequestName(dbName string) string {
	return dbName + MongoPrefix + "-admin-secret-access-req"
}
func getMongoAdminServiceAccountName(dbName string) string {
	return dbName + MongoPrefix + "-admin-service-account"
}

// For Read-write
func getMongoReadWriteRoleName(dbName string) string {
	return dbName + MongoPrefix + "-read-write-role"
}
func getMongoReadWriteSecretAccessRequestName(dbName string) string {
	return dbName + MongoPrefix + "-read-write-secret-access-req"
}
func getMongoReadWriteServiceAccountName(dbName string) string {
	return dbName + MongoPrefix + "-read-write-service-account"
}

func getMongoSecretEngineName(name string) string {
	return name + MongoPrefix + "-secret-engine"
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
