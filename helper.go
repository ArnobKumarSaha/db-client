package main

import (
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
)

func getReplicaCount() *int32 {
	var n int32
	n = 3
	return &n
}

func getStorageTypeName() *string {
	var s string = "storage"
	return &s
}

func createMongoDB() *api.MongoDB {
	return &api.MongoDB{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: mongoDBName,
			Namespace: mongoDBNamespace,
		},
		Spec: api.MongoDBSpec{
			Version:    mongoDBVersion,
			Replicas:   getReplicaCount(),
			ReplicaSet: &api.MongoDBReplicaSet{Name: "rs0'"},
			Storage: &core.PersistentVolumeClaimSpec{
				StorageClassName: getStorageTypeName(),
				AccessModes: []core.PersistentVolumeAccessMode{
					core.ReadWriteOnce,
				},
				Resources: core.ResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceStorage: resource.Quantity{
						},
					},
				},
			},
			/*
				AuthSecret: &core.LocalObjectReference{
					Name: "my-secret",
				},*/
		},
	}
}
