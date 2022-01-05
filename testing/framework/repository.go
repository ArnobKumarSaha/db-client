package framework

import (
	"context"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	store "kmodules.xyz/objectstore-api/api/v1"
	repository "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
)

func (i *Invocation) GetRepositorySpec(dbType string) *repository.Repository {
	ret := &repository.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepositoryName,
			Namespace: i.databaseNamespace,
		},
		Spec: repository.RepositorySpec{
			Backend: store.Backend{
				StorageSecretName: MinioRepositorySecretName,
				S3: &store.S3Spec{
					Endpoint: "http://minio.minio.svc:9000",
					Bucket:   "backup-mongo",
					Prefix:   "standalone", // this will be changed according to dbType
				},
			},
		},
	}
	if dbType == Sharded {
		ret.Spec.Backend.S3.Prefix = "shard"
	} else if dbType == ReplicaSet {
		ret.Spec.Backend.S3.Prefix = "replica"
	}
	return ret
}

const (
	RESTIC_PASSWORD       = "changeit"
	AWS_ACCESS_KEY_ID     = "minio12345"
	AWS_SECRET_ACCESS_KEY = "minio12345"
)

var (
	secret *core.Secret
	repo   *repository.Repository
)

func (i *Invocation) GetRepositorySecretSpec() *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioRepositorySecretName,
			Namespace: i.databaseNamespace,
		},
		Type: core.SecretTypeOpaque,
		Data: map[string][]byte{
			"RESTIC_PASSWORD":       []byte(RESTIC_PASSWORD),
			"AWS_ACCESS_KEY_ID":     []byte(AWS_ACCESS_KEY_ID),
			"AWS_SECRET_ACCESS_KEY": []byte(AWS_SECRET_ACCESS_KEY),
		},
	}
}

func (i *TestOptions) CreateRepository() error {
	// Create the secret first, then Repository itself
	secret = i.GetRepositorySecretSpec()
	var s core.Secret
	err := i.myClient.Get(context.TODO(), types.NamespacedName{
		Namespace: secret.GetNamespace(),
		Name:      secret.GetName(),
	}, &s)
	if errors.IsNotFound(err) {
		err = i.myClient.Create(context.TODO(), secret)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	repo = i.GetRepositorySpec(i.DBType)
	var r repository.Repository
	err = i.myClient.Get(context.TODO(), types.NamespacedName{
		Namespace: repo.GetNamespace(),
		Name:      repo.GetName(),
	}, &r)
	if errors.IsNotFound(err) {
		err = i.myClient.Create(context.TODO(), repo)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return err
}
func (i *TestOptions) DeleteRepository() error {
	// Delete the secret first,  then Repository itself
	err := i.myClient.Delete(context.TODO(), secret)
	if err != nil {
		return err
	}
	err = i.myClient.Delete(context.TODO(), repo)
	return err
}
