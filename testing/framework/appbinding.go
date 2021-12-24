package framework

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcatutil "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

func (i *Invocation) GetAppbindingSpec() *appcatutil.AppBinding {
	return &appcatutil.AppBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
		},
		Spec: appcatutil.AppBindingSpec{
			Type:             "",
			Version:          "",
			ClientConfig:     appcatutil.ClientConfig{},
			Secret:           nil,
			SecretTransforms: nil,
			Parameters:       nil,
		},
	}
}
