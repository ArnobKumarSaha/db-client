package framework

import (
	"context"
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	dbClient "kubedb.dev/db-client-go/mongodb"
	"sigs.k8s.io/controller-runtime/pkg/client"
	repository "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"time"
)

func (i *Invocation) GetBackupConfigurationSpec() *stash.BackupConfiguration {
	return &stash.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BackupConfigName,
			Namespace: i.databaseNamespace,
		},
		Spec: stash.BackupConfigurationSpec{
			BackupConfigurationTemplateSpec: stash.BackupConfigurationTemplateSpec{
				Target: &stash.BackupTarget{
					Ref: stash.TargetRef{
						APIVersion: "appcatalog.appscode.com/v1alpha1",
						Kind:       "AppBinding",
						Name:       MongoDBName,
					},
				},
				Task: stash.TaskRef{
					Name: "mongodb-backup-4.4.6",
				},
			},
			Schedule: "*/1 * * * *",
			Repository: corev1.LocalObjectReference{
				Name: RepositoryName,
			},
			RetentionPolicy: repository.RetentionPolicy{
				Name:     "keep-last-5",
				KeepLast: 5,
				Prune:    true,
			},
		},
	}
}

var (
	bc *stash.BackupConfiguration
)

func (i *TestOptions) CreateBackupConfiguration() error {
	bc = i.GetBackupConfigurationSpec()
	err := i.myClient.Create(context.TODO(), bc)
	if err != nil {
		return err
	}
	return nil
}

func (i *TestOptions) DeleteBackupConfiguration() error {
	err := i.myClient.Delete(context.TODO(), bc)
	if err != nil {
		return err
	}
	return nil
}

func (i *TestOptions) CheckSuccessOfBackupConfiguration() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var backups stash.BackupSessionList
			err := i.myClient.List(context.TODO(), &backups, &client.ListOptions{Namespace: i.databaseNamespace})
			if err != nil {
				return err
			}

			var count = 0
			for _, item := range backups.Items {
				owners := item.OwnerReferences
				for i := 0; i < len(owners); i++ {
					if owners[i].Name == BackupConfigName {
						count++
						// We have found the working BackUpSession, controlled by our BackUpConfiguration
						if item.Status.Phase != stash.BackupSessionSucceeded {
							return errors.New("still backup not not succeeded")
						}
					}
				}
			}
			if count == 0 {
				return errors.New("no backup session has been found controlled by Our backupConfiguration")
			}
			return nil
		},
		time.Minute*6,
		time.Second*10,
	)
}

func (i *TestOptions) InsertData() error {
	ctx := context.TODO()
	//var secs corev1.SecretList
	//err := i.myClient.List(ctx, &secs, &client.ListOptions{
	//	Namespace: i.databaseNamespace,
	//})
	//if err != nil {
	//	return err
	//}
	//fmt.Println("*******************************************************")
	//for _, item := range secs.Items {
	//	fmt.Println(item.Name)
	//}

	var mongo kdm.MongoDB
	err := i.myClient.Get(ctx, types.NamespacedName{
		Namespace: i.databaseNamespace,
		Name:      MongoDBName,
	}, &mongo)

	fmt.Println("phase =================================================================> ", mongo.Status.Phase)

	mongoClient, err := dbClient.NewKubeDBClientBuilder(i.myClient, &mongo).WithContext(ctx).GetMongoClient()
	if err != nil {
		klog.Fatalf("Running MongoDBClient failed. %s", err.Error())
		return err
	}
	defer mongoClient.Close()
	fmt.Println("Lets update now")

	_, err = mongoClient.Database(SchemaName).Collection(CollectionName).InsertOne(context.TODO(), bson.D{
		{"hello", "awesome. Inserting is working from testing-code"},
	})
	return err

}
