package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiv1 "kmodules.xyz/client-go/api/v1"
)

const (
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	InitScriptName              string = "init.js"
	MongoInitScriptPath         string = "/init-scripts"
	MongoPrefix                 string = "-mongo"
	MongoDatabaseNameForEntry   string = "kubedb-system"
	MongoCollectionNameForEntry string = "databases"
)

func (db *MongoDBDatabase) GetMongoInitVolumeNameForPod() string {
	return db.GetName() + "-init-volume"
}
func (db *MongoDBDatabase) GetMongoInitJobName() string {
	return db.GetName() + "-init-job"
}
func (db *MongoDBDatabase) GetMongoInitContainerNameForPod() string {
	return db.GetName() + "-init-container"
}
func (db *MongoDBDatabase) GetMongoRestoreSessionName() string {
	return db.GetName() + "-restore-session"
}

// For Admin
func (db *MongoDBDatabase) GetMongoAdminRoleName() string {
	return db.GetName() + MongoPrefix + "-admin-role"
}
func (db *MongoDBDatabase) GetMongoAdminSecretAccessRequestName() string {
	return db.GetName() + MongoPrefix + "-admin-secret-access-req"
}
func (db *MongoDBDatabase) GetMongoAdminServiceAccountName() string {
	return db.GetName() + MongoPrefix + "-admin-service-account"
}

func (db *MongoDBDatabase) GetMongoSecretEngineName() string {
	return db.GetName() + MongoPrefix + "-secret-engine"
}

func (db *MongoDBDatabase) GetAuthSecretName(dbServerName string) string {
	return dbServerName + "-auth"
}

// ContainsString function is to check and remove string from a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (db *MongoDBDatabase) SetCondition(typ SchemaDatabaseCondition, sts corev1.ConditionStatus, reason SchemaDatabaseReason, msg string) []apiv1.Condition {
	gen := int64(1)
	for i := range db.Status.Conditions {
		c := db.Status.Conditions[i]
		if c.Type == string(typ) {
			gen = c.ObservedGeneration
			if c.Status != sts {
				gen += 1
			}
		}
	}
	db.Status.Conditions = apiv1.SetCondition(db.Status.Conditions, apiv1.Condition{
		Type:               string(typ),
		Status:             sts,
		Reason:             string(reason),
		ObservedGeneration: gen,
		Message:            msg,
	})
	return db.Status.Conditions
}
