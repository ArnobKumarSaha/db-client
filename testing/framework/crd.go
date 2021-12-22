package framework

import (
	"context"
	"errors"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
			var schema smv1a1.MongoDBDatabase
			err := i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      "sample",
				Namespace: i.Namespace(),
			}, &schema)
			if err != nil {
				return err
			}

			var vault kvm_server.VaultServer
			err = i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      "vault",
				Namespace: i.Namespace(),
			}, &vault)
			if err != nil {
				return err
			}

			var mongo kdm.MongoDB
			err = i.myClient.Get(context.TODO(), types.NamespacedName{
				Name:      "mng-shrd",
				Namespace: i.Namespace(),
			}, &mongo)
			if err != nil {
				return err
			}

			if mongo.Status.Phase != kdm.DatabasePhaseReady || vault.Status.Phase != kvm_server.VaultServerPhaseReady {
				return errors.New("MongoDB or Vault server is not Ready yet")
			}

			return nil
		},
		time.Minute*3,
		time.Second*20,
	)
}

func (i *Invocation) CheckCompletenessOfInit() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var pods corev1.PodList
			err := i.myClient.List(context.TODO(), &pods, &client.ListOptions{Namespace: i.Namespace()})
			if err != nil {
				return err
			}
			// Looping through the pods to find out the InitPod that was created by the InitJob
			// And return nil if it is already succeeded, else return error
			for _, pod := range pods.Items {
				for i := 0; i < len(pod.OwnerReferences); i++ {
					ref := pod.OwnerReferences[i]
					if ref.Name != "sample-init-job" {
						continue
					}
					if pod.Status.Phase != corev1.PodSucceeded {
						return errors.New("pod is not succeeded yet")
					} else {
						return nil
					}
				}
			}
			return nil
		},
		time.Minute*1,
		time.Second*10,
	)
}
