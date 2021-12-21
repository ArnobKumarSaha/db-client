package schema

import (
	"context"
	"fmt"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	kutil "kmodules.xyz/client-go"
	kmapi "kmodules.xyz/client-go/api/v1"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_apis "kubevault.dev/apimachinery/apis"
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPhaseFromCondition(conditions []kmapi.Condition) schemav1alpha1.SchemaDatabasePhase {
	// Set the SchemeDatabasePhase to 'running', if any of them is 'Not ready'
	if !kmapi.IsConditionTrue(conditions, string(schemav1alpha1.SchemaDatabaseConditionDBReady)) {
		return schemav1alpha1.SchemaDatabasePhaseRunning
	}

	if !kmapi.IsConditionTrue(conditions, string(schemav1alpha1.SchemaDatabaseConditionVaultReady)) {
		return schemav1alpha1.SchemaDatabasePhaseRunning
	}

	if !kmapi.IsConditionTrue(conditions, string(schemav1alpha1.SchemaDatabaseConditionSecretEngineReady)) {
		return schemav1alpha1.SchemaDatabasePhaseRunning
	}

	if !kmapi.IsConditionTrue(conditions, string(schemav1alpha1.SchemaDatabaseConditionPostgresRoleReady)) {
		return schemav1alpha1.SchemaDatabasePhaseRunning
	}

	if !kmapi.IsConditionTrue(conditions, string(schemav1alpha1.SchemaDatabaseConditionSecretAccessRequestReady)) {
		return schemav1alpha1.SchemaDatabasePhaseRunning
	}

	if kmapi.HasCondition(conditions, string(schemav1alpha1.SchemaDatabaseConditionJobCompleted)) && !kmapi.IsConditionTrue(conditions, string(schemav1alpha1.SchemaDatabaseConditionJobCompleted)) {
		return schemav1alpha1.SchemaDatabasePhaseRunning
	}
	return schemav1alpha1.SchemaDatabasePhaseSucceeded
}

func CheckVaultConditions(vaultServer *kvm_server.VaultServer) bool {
	cond := true
	cond = cond && kmapi.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.AllReplicasAreReady)
	cond = cond && kmapi.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.VaultServerAcceptingConnection)
	cond = cond && kmapi.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.VaultServerInitialized)
	cond = cond && kmapi.IsConditionTrue(vaultServer.Status.Conditions, kvm_apis.VaultServerUnsealed)
	return cond
}

func CheckSecretEngineConditions(secretEng *kvm_engine.SecretEngine) bool {
	return kmapi.IsConditionTrue(secretEng.Status.Conditions, kmapi.ConditionAvailable)
}

func CheckMongoDBRoleConditions(dbRole *kvm_engine.MongoDBRole) bool {
	cond := true
	cond = cond && kmapi.IsConditionTrue(dbRole.Status.Conditions, kmapi.ConditionAvailable)
	return cond
}

func CheckPostgresRoleConditions(dbRole *kvm_engine.PostgresRole) bool {
	cond := true
	cond = cond && kmapi.IsConditionTrue(dbRole.Status.Conditions, kmapi.ConditionAvailable)
	return cond
}

func CheckMysqlRoleConditions(dbRole *kvm_engine.MySQLRole) bool {
	cond := true
	cond = cond && kmapi.IsConditionTrue(dbRole.Status.Conditions, kmapi.ConditionAvailable)
	return cond
}

func CheckSecretAccessRequestConditions(acrObj *kvm_engine.SecretAccessRequest) bool {
	cond := true
	cond = cond && kmapi.IsConditionTrue(acrObj.Status.Conditions, kmapi.ConditionAvailable)
	cond = cond && kmapi.IsConditionTrue(acrObj.Status.Conditions, kmapi.ConditionRequestApproved)
	return cond
}

func CheckJobConditions(job *batch.Job) bool {
	cond := true
	cond = cond && IsJobConditionTrue(job.Status.Conditions, batch.JobComplete)
	return cond
}

func IsJobConditionTrue(conditions []batch.JobCondition, condType batch.JobConditionType) bool {
	for i := range conditions {
		if conditions[i].Type == condType && conditions[i].Status == core.ConditionTrue {
			return true
		}
	}
	return false
}

type TransformFunc func(obj client.Object, createOp bool) client.Object

func PatchStatus(c client.Client, obj client.Object, transform TransformFunc, opts ...client.PatchOption) (client.Object, kutil.VerbType, error) {
	key := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	err := c.Get(context.TODO(), key, obj)
	if errors.IsNotFound(err) {
		fmt.Println("+++++++++++++++++++++ IsNotFoundError from PatchStatus")
	}
	if err != nil {
		fmt.Println("+++++++++++++++++++++ Error from PatchStatus")
		return nil, kutil.VerbUnchanged, err
	}

	//patch := client.MergeFrom(obj)

	obj = transform(obj.DeepCopyObject().(client.Object), false)
	//err = c.Status().Patch(context.TODO(), obj, patch, opts...)
	err = c.Status().Update(context.TODO(), obj)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return obj, kutil.VerbPatched, nil
}

func SetCondition(database *schemav1alpha1.MongoDBDatabase, typ schemav1alpha1.SchemaDatabaseCondition, sts core.ConditionStatus, reason schemav1alpha1.SchemaDatabaseReason, msg string) {
	gen := int64(1)
	for i := range database.Status.Conditions {
		c := database.Status.Conditions[i]
		if c.Type == string(typ) && c.Status != sts {
			gen += 1
		}
	}
	database.Status.Conditions = kmapi.SetCondition(database.Status.Conditions, kmapi.Condition{
		Type:               string(typ),
		Status:             sts,
		Reason:             string(reason),
		ObservedGeneration: gen,
		Message:            msg,
	})
}
