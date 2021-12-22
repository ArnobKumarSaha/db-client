package framework

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
)

var (
	cm *corev1.ConfigMap
)

func (i *Invocation) GetSchemaMongoDBDatabaseSpec() *smv1a1.MongoDBDatabase {
	return &smv1a1.MongoDBDatabase{
		ObjectMeta: meta.ObjectMeta{
			Name:      "sample",
			Namespace: i.Namespace(),
		},
		Spec: smv1a1.MongoDBDatabaseSpec{
			DatabaseRef: apiv1.ObjectReference{
				Name:      "mng-shrd",
				Namespace: i.Namespace(),
			},
			VaultRef: apiv1.ObjectReference{
				Name:      "vault",
				Namespace: i.Namespace(),
			},
			DatabaseSchema: smv1a1.DatabaseSchema{
				Name: "mydb",
			},
			Subjects: []smv1a1.Subject{
				{
					Name:      "sa_name",
					Namespace: i.Namespace(),
					SubjectKind: meta.TypeMeta{
						Kind:       "ServiceAccount",
						APIVersion: "v1",
					},
				},
			},
			Init: &smv1a1.InitSpec{
				Initialized: false,
				Script: &smv1a1.ScriptSourceSpec{
					ScriptPath: "/etc/config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-cm",
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
			},
			/*Restore: &smv1a1.RestoreRef{
				Repository: apiv1.ObjectReference{
					Name:      "local-repo",
					Namespace: i.Namespace(),
				},
				Snapshot: "latest",
			},*/
			DeletionPolicy: smv1a1.DeletionPolicyDelete,
			AutoApproval:   true,
		},
	}
}

func (i *Invocation) CreateMongoDBDatabaseSchema(m *smv1a1.MongoDBDatabase) error {
	// Create the configmap first, then MongoDbDatabase itself
	cm = i.GetConfigMapSpec()
	err := i.myClient.Create(context.TODO(), cm)
	if err != nil {
		return err
	}

	err = i.myClient.Create(context.TODO(), m)
	return err
}
func (i *Invocation) DeleteMongoDBDatabaseSchema(m *smv1a1.MongoDBDatabase) error {
	// Delete the configmap first,  then MongoDbDatabase itself
	err := i.myClient.Delete(context.TODO(), cm)
	if err != nil {
		return err
	}
	err = i.myClient.Delete(context.TODO(), m)
	return err
}
