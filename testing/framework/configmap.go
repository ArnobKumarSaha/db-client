package framework

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *Invocation) GetConfigMapSpec() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: i.Namespace(),
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
	err := i.myClient.Create(context.TODO(), cm)
	return err
}
func (i *Invocation) DeleteConfigMap() error {
	err := i.myClient.Delete(context.TODO(), cm)
	return err
}
