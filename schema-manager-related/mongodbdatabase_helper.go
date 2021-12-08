package schema

import (
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
)

const (
	KubeDBSchemaManagerFinalizerName string = "v1alpha1.kubedb.dev/finalizer"
	ResourceKindMongoDBDatabase      string = "MongoDBDatabase"
	MongoInitContainerNameForPod     string = "mongo-init-container"
	MongoInitJobName                 string = "init-job"
	MongoInitVolumeNameForPod        string = "init-volume"
	InitScriptName                   string = "init.js"
	MongoInitScriptPath              string = "/init-scripts"
	MongoPrefix                      string = "-mongo"
	MongoDatabaseNameForEntry        string = "kubedb-system"
	MongoCollectionNameForEntry      string = "databases"
)

const (
	SecretEnginePhaseSuccess    kvm_engine.SecretEnginePhase = "Success"
	SecretEnginePhaseProcessing kvm_engine.SecretEnginePhase = "Processing"
)

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
