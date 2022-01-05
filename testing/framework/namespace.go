package framework

import (
	"context"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (f *Framework) Namespace() string {
	return f.schemaNamespace
}

func (f *Framework) CreateNamespaces() error {
	obj := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: SchemaNamespace, //f.schemaNamespace,
		},
	}
	_, err := f.kubeClient.CoreV1().Namespaces().Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	obj.Name = DatabaseNamespace //f.databaseNamespace
	_, err = f.kubeClient.CoreV1().Namespaces().Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	obj.Name = VaultNamespace //f.vaultNamespace
	_, err = f.kubeClient.CoreV1().Namespaces().Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func (f *Framework) DeleteNamespaces() error {
	err := f.kubeClient.CoreV1().Namespaces().Delete(context.TODO(), VaultNamespace /*f.vaultNamespace*/, meta_util.DeleteInForeground())
	if err != nil {
		return err
	}

	err = f.kubeClient.CoreV1().Namespaces().Delete(context.TODO(), DatabaseNamespace /*f.databaseNamespace*/, meta_util.DeleteInForeground())
	if err != nil {
		return err
	}

	err = f.kubeClient.CoreV1().Namespaces().Delete(context.TODO(), SchemaNamespace /*f.schemaNamespace*/, meta_util.DeleteInForeground())
	return err
}
