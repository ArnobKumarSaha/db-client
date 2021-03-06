package schema

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1util "kmodules.xyz/client-go/api/v1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_apis "kubevault.dev/apimachinery/apis"
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
)

/*
phase waiting (handled in controller.go)
---------- MongoDb or VaultServer is not ready --------
phase processing  --> (Can be Ignored when ensuring the entry)
---------- SecretAccessRequest is approved  ---------
phase running
    /     \
   /       \
 /          \
succeeded  failed
*/

func GetPhaseFromCondition(conditions []apiv1util.Condition, toRestore bool) smv1a1.SchemaDatabasePhase {
	// Set the SchemeDatabasePhase to 'waiting', if any of MongoDB-VaultServer is 'Not ready'
	if !apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionVaultReady)) ||
		!apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionDBReady)) {
		return smv1a1.SchemaDatabasePhaseWaiting
	}

	// Set the SchemeDatabasePhase to 'processing', if any of SecretEngine-MongoDBRole-SecretAccessRequest is 'Not ready'
	if !apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionSecretEngineReady)) ||
		!apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionMongoDBRoleReady)) ||
		!apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionSecretAccessRequestReady)) {
		return smv1a1.SchemaDatabasePhaseProcessing
	}

	// Now differentiate among 'running', 'succeeded' & 'failed'
	if toRestore {
		if !apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionRepositoryFound)) {
			return smv1a1.SchemaDatabasePhaseRunning
		}
		if !apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionAppbindingFound)) {
			return smv1a1.SchemaDatabasePhaseRunning
		}
		if !apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionRestoreSession)) {
			// We are here means, RestoreSession phase equals to neither succeed nor failed
			return smv1a1.SchemaDatabasePhaseRunning
		}
	} else {
		if apiv1util.HasCondition(conditions, string(smv1a1.SchemaDatabaseConditionJobCompleted)) && !apiv1util.IsConditionTrue(conditions, string(smv1a1.SchemaDatabaseConditionJobCompleted)) {
			return smv1a1.SchemaDatabasePhaseRunning
		}
	}
	for i := range conditions {
		if conditions[i].Type == string(smv1a1.SchemaDatabaseConditionRestoreSession) {
			if conditions[i].Reason == string(smv1a1.SchemaDatabaseReasonRestoreSessionSFailed) {
				return smv1a1.SchemaDatabasePhaseFailed
			} else if conditions[i].Reason == string(smv1a1.SchemaDatabaseReasonRestoreSessionSucceed) {
				return smv1a1.SchemaDatabasePhaseSucceeded
			}
		}
	}
	return smv1a1.SchemaDatabasePhaseSucceeded
}

func CheckVaultConditions(vaultServer *kvm_server.VaultServer) bool {
	cond := true
	cond = cond && apiv1util.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.AllReplicasAreReady)
	cond = cond && apiv1util.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.VaultServerAcceptingConnection)
	cond = cond && apiv1util.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.VaultServerInitialized)
	cond = cond && apiv1util.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.VaultServerUnsealed)
	return cond
}

func CheckMongoDBConditions(mongo *kdm.MongoDB) bool {
	cond := true
	cond = cond && apiv1util.IsConditionTrue(mongo.Status.Conditions, kdm.DatabaseAcceptingConnection)
	cond = cond && apiv1util.IsConditionTrue(mongo.Status.Conditions, kdm.DatabaseReady)
	cond = cond && apiv1util.IsConditionTrue(mongo.Status.Conditions, kdm.DatabaseProvisioned)
	return cond
}

func CheckSecretEngineConditions(secretEng *kvm_engine.SecretEngine) bool {
	return apiv1util.IsConditionTrue(secretEng.Status.Conditions, apiv1util.ConditionAvailable)
}

func CheckMongoDBRoleConditions(dbRole *kvm_engine.MongoDBRole) bool {
	cond := true
	cond = cond && apiv1util.IsConditionTrue(dbRole.Status.Conditions, apiv1util.ConditionAvailable)
	return cond
}

func CheckPostgresRoleConditions(dbRole *kvm_engine.PostgresRole) bool {
	cond := true
	cond = cond && apiv1util.IsConditionTrue(dbRole.Status.Conditions, apiv1util.ConditionAvailable)
	return cond
}

func CheckMysqlRoleConditions(dbRole *kvm_engine.MySQLRole) bool {
	cond := true
	cond = cond && apiv1util.IsConditionTrue(dbRole.Status.Conditions, apiv1util.ConditionAvailable)
	return cond
}

func CheckSecretAccessRequestConditions(acrObj *kvm_engine.SecretAccessRequest) bool {
	cond := true
	cond = cond && apiv1util.IsConditionTrue(acrObj.Status.Conditions, apiv1util.ConditionAvailable)
	cond = cond && apiv1util.IsConditionTrue(acrObj.Status.Conditions, apiv1util.ConditionRequestApproved)
	return cond
}

func CheckRestoreSessionPhaseSucceed(res *stash.RestoreSession) bool {
	return res.Status.Phase == stash.RestoreSucceeded
}

func CheckRestoreSessionPhaseFailed(res *stash.RestoreSession) bool {
	return res.Status.Phase == stash.RestoreFailed
}

func CheckJobConditions(job *batchv1.Job) bool {
	cond := true
	cond = cond && IsJobConditionTrue(job.Status.Conditions, batchv1.JobComplete)
	return cond
}

func IsJobConditionTrue(conditions []batchv1.JobCondition, condType batchv1.JobConditionType) bool {
	for i := range conditions {
		if conditions[i].Type == condType && conditions[i].Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
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
