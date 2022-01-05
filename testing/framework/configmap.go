package framework

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (i *Invocation) GetConfigMapSpec() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName,
			Namespace: i.schemaNamespace,
		},
		Data: map[string]string{
			"hello":   "world",
			"init.js": "use mydb;\n    db.people.insert({\"firstname\" : \"kubernetes\", \"lastname\" : \"database\" });",
		},
	}
}

var (
	cm *corev1.ConfigMap
)

func (i *Invocation) CreateConfigMap() error {
	cm = i.GetConfigMapSpec()
	var c corev1.ConfigMap
	err := i.myClient.Get(context.TODO(), types.NamespacedName{
		Namespace: cm.GetNamespace(),
		Name:      cm.GetName(),
	}, &c)
	if errors.IsNotFound(err) {
		err = i.myClient.Create(context.TODO(), cm)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
func (i *Invocation) DeleteConfigMap() error {
	err := i.myClient.Delete(context.TODO(), cm)
	return err
}
