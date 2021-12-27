package framework

import (
	"context"
	"errors"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apiv1 "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	"time"
)

func (i *Invocation) GetSchemaMongoDBDatabaseSpec(opts ...*SchemaOptions) *smv1a1.MongoDBDatabase {
	retObj := &smv1a1.MongoDBDatabase{
		ObjectMeta: meta.ObjectMeta{
			Name:      MongoDBDatabaseSchemaName,
			Namespace: i.Namespace(),
		},
		Spec: smv1a1.MongoDBDatabaseSpec{
			DatabaseRef: apiv1.ObjectReference{
				Name:      MongoDBName,
				Namespace: i.Namespace(),
			},
			VaultRef: apiv1.ObjectReference{
				Name:      VaultName,
				Namespace: i.Namespace(),
			},
			DatabaseSchema: smv1a1.DatabaseSchema{
				Name: SchemaName,
			},
			Subjects: []smv1a1.Subject{
				{
					Name:      SubjectName,
					Namespace: i.Namespace(),
					SubjectKind: meta.TypeMeta{
						Kind:       "ServiceAccount",
						APIVersion: "v1",
					},
				},
			},
			DeletionPolicy: smv1a1.DeletionPolicyDelete,
		},
	}

	for _, opt := range opts {
		retObj.Spec.AutoApproval = opt.AutoApproval
		if opt.ToRestore {
			retObj.Spec.Restore = &smv1a1.RestoreRef{
				Repository: apiv1.ObjectReference{
					Name:      RepositoryName,
					Namespace: i.Namespace(),
				},
				Snapshot: "latest",
			}
		} else {
			retObj.Spec.Init = &smv1a1.InitSpec{
				Initialized: false,
				Script: &smv1a1.ScriptSourceSpec{
					ScriptPath: "/etc/config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: ConfigMapName,
							},
						},
					},
				},
				PodTemplate: &ofst.PodTemplateSpec{
					Spec: ofst.PodSpec{
						Env: []corev1.EnvVar{
							{
								Name:  "HAVE_A_TRY",
								Value: "whoo! It works",
							},
						},
					},
				},
			}
		}
	}
	return retObj
}

func (i *TestOptions) CreateMongoDBDatabaseSchema() error {
	// Create the configmap or repository first.. based on if restore is enabled or not, then create MongoDbDatabase itself
	if i.ToRestore {
		err := i.CreateRepository()
		if err != nil {
			return err
		}
	} else {
		err := i.CreateConfigMap()
		if err != nil {
			return err
		}
	}
	err := i.myClient.Create(context.TODO(), i.SchemaDatabase)
	return err
}

func (i *TestOptions) DeleteMongoDBDatabaseSchema() error {
	// Delete the configmap or repository first.. based on if restore is enabled or not, then delete MongoDbDatabase itself
	if i.ToRestore {
		err := i.DeleteRepository()
		if err != nil {
			return err
		}
	} else {
		err := i.DeleteConfigMap()
		if err != nil {
			return err
		}
	}
	err := i.myClient.Delete(context.TODO(), i.SchemaDatabase)
	return err
}

func (i *TestOptions) CheckSuccessOfSchema() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			var obj smv1a1.MongoDBDatabase
			err := i.myClient.Get(context.TODO(), types.NamespacedName{
				Namespace: i.Namespace(),
				Name:      MongoDBDatabaseSchemaName,
			}, &obj)
			if err != nil {
				return err
			}
			if obj.Status.Phase != smv1a1.SchemaDatabasePhaseSucceeded {
				return errors.New("still not succeeded")
			}
			return nil
		},
		time.Minute*6,
		time.Second*10,
	)
}
