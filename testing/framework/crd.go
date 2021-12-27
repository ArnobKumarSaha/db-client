package framework

import (
	"context"
	"errors"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func (f *Framework) EventuallyCRD() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var schemas smv1a1.MongoDBDatabaseList

			err := f.myClient.List(context.TODO(), &schemas, &client.ListOptions{Namespace: f.Namespace()})
			if err != nil {
				return err //errors.New("CRD Instances is not ready")
			}
			return nil
		},
		time.Minute*1,
		time.Second*10,
	)
}

func (i *Invocation) CheckReadiness() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var vault kvm_server.VaultServer
			err := i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      VaultName,
				Namespace: i.Namespace(),
			}, &vault)
			if err != nil {
				return err
			}

			var mongo kdm.MongoDB
			err = i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      MongoDBName,
				Namespace: i.Namespace(),
			}, &mongo)
			if err != nil {
				return err
			}

			if (mongo.Status.Phase == kdm.DatabasePhaseReady || mongo.Status.Phase == kdm.DatabasePhaseDataRestoring) && vault.Status.Phase == kvm_server.VaultServerPhaseReady {
				return nil
			}

			return errors.New("MongoDB or Vault server is not Ready yet")
		},
		time.Minute*6,
		time.Second*20,
	)
}

func (i *TestOptions) CleanUpEverything() error {
	err := i.DeleteMongoDB()
	if err != nil {
		return err
	}
	err = i.DeleteVaultServer()
	if err != nil {
		return err
	}

	err = i.DeleteRunnerJob()
	if err != nil {
		return err
	}
	return nil
}

func (i *TestOptions) CheckIfEverythingIsCleaned() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var ret error = nil
			var vault kvm_server.VaultServer
			err := i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      VaultName,
				Namespace: i.Namespace(),
			}, &vault)
			if !kerrors.IsNotFound(err) {
				return errors.New("vault is not deleted yet")
			}

			var mongo kdm.MongoDB
			err = i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      MongoDBName,
				Namespace: i.Namespace(),
			}, &mongo)
			if !kerrors.IsNotFound(err) {
				return errors.New("MongoDb is not deleted yet")
			}

			var job batchv1.Job
			err = i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      RunnerJobName,
				Namespace: i.Namespace(),
			}, &job)
			if !kerrors.IsNotFound(err) {
				return errors.New("InitJob is not deleted yet")
			}
			return ret
		},
		time.Minute*1,
		time.Second*10,
	)
}

func (i *TestOptions) WaitForMongoDBDBDDatabaseAndDependantsCleanup() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var db smv1a1.MongoDBDatabase
			err := i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      MongoDBDatabaseSchemaName,
				Namespace: i.Namespace(),
			}, &db)
			if !kerrors.IsNotFound(err) {
				return errors.New("MongoDBDatabase is not deleted yet")
			}
			return nil
		},
		time.Minute*2,
		time.Second*10,
	)
}
