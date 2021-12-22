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

func (i *Invocation) CreateConfigMap(c *corev1.ConfigMap) error {
	err := i.myClient.Create(context.TODO(), c)
	return err
}
func (i *Invocation) DeleteConfigMap(c *corev1.ConfigMap) error {
	err := i.myClient.Delete(context.TODO(), c)
	return err
}
