package framework

import (
	"context"
	"fmt"
	. "github.com/onsi/gomega"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func (f *Framework) EventuallyCRD() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var schemas smv1a1.MongoDBDatabaseList

			err := f.myClient.List(context.TODO(), &schemas, &client.ListOptions{Namespace: f.Namespace()})
			fmt.Println("schemas : ", schemas, "ns : ", f.Namespace())
			if err != nil {
				return err //errors.New("CRD Instances is not ready")
			}
			return nil
		},
		time.Minute*1,
		time.Second*10,
	)
}
