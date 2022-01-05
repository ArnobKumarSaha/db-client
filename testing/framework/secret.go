package framework

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (i *Invocation) GetSecretSpec() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretName,
			Namespace: i.schemaNamespace,
		},
		Data: map[string][]byte{
			"hello":   []byte("world"),
			"init.js": []byte("use mydb;\n    db.people.insert({\"firstname\" : \"kubernetes\", \"lastname\" : \"database\" });"),
		},
	}
}

var (
	sec *corev1.Secret
)

func (i *Invocation) CreateSecret() error {
	sec = i.GetSecretSpec()
	var s corev1.Secret
	err := i.myClient.Get(context.TODO(), types.NamespacedName{
		Namespace: sec.GetNamespace(),
		Name:      sec.GetName(),
	}, &s)
	if errors.IsNotFound(err) {
		err = i.myClient.Create(context.TODO(), sec)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
func (i *Invocation) DeleteSecret() error {
	err := i.myClient.Delete(context.TODO(), sec)
	return err
}
