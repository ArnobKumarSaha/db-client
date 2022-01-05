package framework

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	mona "kmodules.xyz/monitoring-agent-api/api/v1"
	kvm "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
)

func (i *Invocation) GetVaultServerSpec() *kvm.VaultServer {
	return &kvm.VaultServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      VaultName,
			Namespace: i.vaultNamespace,
		},
		Spec: kvm.VaultServerSpec{
			Version:  "1.8.2",
			Replicas: func(i int32) *int32 { return &i }(3),
			Backend: kvm.BackendStorageSpec{
				Raft: &kvm.RaftSpec{
					Path: "/vault/data",
					Storage: &corev1.PersistentVolumeClaimSpec{
						StorageClassName: func(s string) *string { return &s }("standard"),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"storage": resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			Unsealer: &kvm.UnsealerSpec{
				SecretShares:    5,
				SecretThreshold: 3,
				Mode: kvm.ModeSpec{
					KubernetesSecret: &kvm.KubernetesSecretSpec{
						SecretName: "vault-keys",
					},
				},
			},
			AuthMethods: []kvm.AuthMethod{
				{
					Type: "kubernetes",
					Path: "kubernetes",
				},
			},
			Monitor: &mona.AgentSpec{
				Agent: mona.VendorPrometheus,
				Prometheus: &mona.PrometheusSpec{
					Exporter: mona.PrometheusExporterSpec{
						Resources: corev1.ResourceRequirements{},
					},
				},
			},
			TerminationPolicy: kvm.TerminationPolicyWipeOut,
			AllowedSecretEngines: &kvm.AllowedSecretEngines{
				Namespaces: &kvm.SecretEngineNamespaces{
					From: func(s kvm.FromNamespaces) *kvm.FromNamespaces { return &s }(kvm.NamespacesFromAll),
				},
				SecretEngines: []kvm.SecretEngineType{
					kvm.SecretEngineTypeMongoDB,
				},
			},
		},
	}
}

func (i *TestOptions) CreateVaultServer() error {
	err := i.myClient.Create(context.TODO(), i.Vault)
	return err
}

func (i *TestOptions) DeleteVaultServer() error {
	err := i.myClient.Delete(context.TODO(), i.Vault)
	// Get & delete the vault secret manually
	var v_sec corev1.Secret
	err = i.myClient.Get(context.TODO(), types.NamespacedName{
		Namespace: i.vaultNamespace,
		Name:      VaultName + "-keys",
	}, &v_sec)
	err = i.myClient.Delete(context.TODO(), &v_sec)
	return err
}
