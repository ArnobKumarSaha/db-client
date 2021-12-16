package schema

import (
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
)

const (
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	InitScriptName              string = "init.js"
	MongoInitScriptPath         string = "/init-scripts"
	MongoPrefix                 string = "-mongo"
	MongoDatabaseNameForEntry   string = "kubedb-system"
	MongoCollectionNameForEntry string = "databases"
)

func getMongoInitVolumeNameForPod(dbName string) string {
	return dbName + "-init-volume"
}
func getMongoInitJobName(dbName string) string {
	return dbName + "-init-job"
}
func getMongoInitContainerNameForPod(dbName string) string {
	return dbName + "-init-container"
}
func getMongoRestoreSessionName(dbName string) string {
	return dbName + "-restore-session"
}

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

func getAuthSecretName(name string) string {
	return name + "-auth"
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
