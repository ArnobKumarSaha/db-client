package v1alpha1

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
	SchemaDatabasePhaseRunning   SchemaDatabasePhase = "Running"
	SchemaDatabasePhaseSucceeded SchemaDatabasePhase = "Succeeded"
	SchemaDatabasePhaseFailed    SchemaDatabasePhase = "Failed"
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
)
