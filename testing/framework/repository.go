package framework

import (
	"context"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	repository "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
)

func (i *Invocation) GetRepositorySpec(dbType string) *repository.Repository {
	ret := &repository.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepositoryName,
			Namespace: i.Namespace(),
		},
		Spec: repository.RepositorySpec{
			Backend: store.Backend{
				StorageSecretName: RepositorySecretName,
				S3: &store.S3Spec{
					Endpoint: "https://us-southeast-1.linodeobjects.com",
					Bucket:   "backup-mongo",
					Prefix:   "alone", // this will be changed according to dbType
					Region:   "us-southeast-1",
				},
			},
		},
	}
	if dbType == Sharded {
		ret.Spec.Backend.S3.Prefix = "demo"
	} else if dbType == ReplicaSet {
		ret.Spec.Backend.S3.Prefix = "replica"
	}
	return ret
}

const (
	RESTIC_PASSWORD       = "changeit"
	AWS_ACCESS_KEY_ID     = "FXM5IHHQN4YKR0DGHWMK"
	AWS_SECRET_ACCESS_KEY = "aMo34LD11kUaIzkYPsdlkkArxDDCWZUweI4Q887g"
)

var (
	secret *core.Secret
	repo   *repository.Repository
)

func (i *Invocation) GetRepositorySecretSpec() *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepositorySecretName,
			Namespace: i.Namespace(),
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
	err := i.myClient.Create(context.TODO(), secret)
	if err != nil {
		return err
	}

	repo = i.GetRepositorySpec(i.DBType)
	err = i.myClient.Create(context.TODO(), repo)
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
