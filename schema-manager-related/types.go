package v1alpha1

import (
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
)

type DeletionPolicy string

const (
	// DeletionPolicyDelete Deletes database pods, service, pvcs and stash backup data.
	DeletionPolicyDelete DeletionPolicy = "Delete"
	// DeletionPolicyDoNotDelete Rejects attempt to delete database using ValidationWebhook.
	DeletionPolicyDoNotDelete DeletionPolicy = "DoNotDelete"
)

type SchemaDatabasePhase string

const (
	// Phase 'Running' if it is currently working with the objects, on which it is dependant on.
	// 'succeeded' if all of them is successful, 'failed' if any of them is failed
	SchemaDatabasePhaseWaiting    SchemaDatabasePhase = "waiting"
	SchemaDatabasePhaseProcessing SchemaDatabasePhase = "Processing"
	SchemaDatabasePhaseRunning    SchemaDatabasePhase = "Running"
	SchemaDatabasePhaseIgnored    SchemaDatabasePhase = "Ignored"
	SchemaDatabasePhaseSucceeded  SchemaDatabasePhase = "Succeeded"
	SchemaDatabasePhaseFailed     SchemaDatabasePhase = "Failed"
)

type SchemaDatabaseCondition string

const (
	SchemaDatabaseConditionDBReady                  SchemaDatabaseCondition = "DatabaseReady"
	SchemaDatabaseConditionVaultReady               SchemaDatabaseCondition = "VaultReady"
	SchemaDatabaseConditionSecretEngineReady        SchemaDatabaseCondition = "SecretEngineReady"
	SchemaDatabaseConditionSecretAccessRequestReady SchemaDatabaseCondition = "SecretAccessRequestReady"
	SchemaDatabaseConditionJobCompleted             SchemaDatabaseCondition = "JobCompleted"
	// Database Related
	SchemaDatabaseConditionMongoDBRoleReady  SchemaDatabaseCondition = "MongoDBRoleReady"
	SchemaDatabaseConditionPostgresRoleReady SchemaDatabaseCondition = "PostgresRoleReady"
	SchemaDatabaseConditionMysqlRoleReady    SchemaDatabaseCondition = "MysqlRoleReady"
	SchemaDatabaseConditionMariaDBRoleReady  SchemaDatabaseCondition = "MariaDBRoleReady"
	// Stash related
	SchemaDatabaseConditionRepositoryFound SchemaDatabaseCondition = "RepositoryFound"
	SchemaDatabaseConditionAppbindingFound SchemaDatabaseCondition = "AppbindingFound"
	SchemaDatabaseConditionRestoreSession  SchemaDatabaseCondition = "RestoreSession"
)

type SchemaDatabaseReason string

const (
	SchemaDatabaseReasonDBReady                  SchemaDatabaseReason = "CheckDBIsReady"
	SchemaDatabaseReasonVaultReady               SchemaDatabaseReason = "CheckVaultIsReady"
	SchemaDatabaseReasonSecretEngineReady        SchemaDatabaseReason = "CheckSecretEngineIsReady"
	SchemaDatabaseReasonSecretAccessRequestReady SchemaDatabaseReason = "CheckSecretAccessRequestIsReady"
	SchemaDatabaseReasonJobCompleted             SchemaDatabaseReason = "CheckJobIsCompleted"
	// Database Related
	SchemaDatabaseReasonMongoDBRoleReady  SchemaDatabaseReason = "CheckMongoDBRoleIsReady"
	SchemaDatabaseReasonPostgresRoleReady SchemaDatabaseReason = "CheckPostgresRoleIsReady"
	SchemaDatabaseReasonMysqlRoleReady    SchemaDatabaseReason = "CheckMysqlRoleIsReady"
	SchemaDatabaseReasonMariaDbRoleReady  SchemaDatabaseReason = "CheckMariaDBRoleIsReady"
	// Stash related
	SchemaDatabaseReasonRepositoryFound       SchemaDatabaseReason = "RepositoryIsFound"
	SchemaDatabaseReasonAppbindingFound       SchemaDatabaseReason = "AppbindingIsFound"
	SchemaDatabaseReasonRestoreSessionSucceed SchemaDatabaseReason = "RestoreSessionIsSucceeded"
	SchemaDatabaseReasonRestoreSessionSFailed SchemaDatabaseReason = "RestoreSessionIsFailed"

	SchemaDatabaseAutoApprovalReason SchemaDatabaseReason = "ApprovedBySchemaManager"
)

const (
	SecretEnginePhaseSuccess    kvm_engine.SecretEnginePhase = "Success"
	SecretEnginePhaseProcessing kvm_engine.SecretEnginePhase = "Processing"
)
