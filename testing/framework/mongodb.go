package framework

import (
	"context"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientutil "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
)

func (i *Invocation) GetMongoDBSpec(opts ...*DBOptions) *kdm.MongoDB {
	podTmpl := ofst.PodTemplateSpec{
		Spec: ofst.PodSpec{
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("100Mi"),
				},
			},
		},
	}
	storage := &core.PersistentVolumeClaimSpec{
		Resources: core.ResourceRequirements{
			Requests: core.ResourceList{
				"storage": resource.MustParse("100Mi"),
			},
		},
		StorageClassName: func(s string) *string { return &s }("standard"),
		AccessModes: []core.PersistentVolumeAccessMode{
			core.ReadWriteOnce,
		},
	}

	retDB := &kdm.MongoDB{
		ObjectMeta: meta.ObjectMeta{
			Name:      MongoDBName,
			Namespace: i.databaseNamespace,
		},
		Spec: kdm.MongoDBSpec{
			Version: MongoDBVersion,
		},
	}
	for _, opt := range opts {
		// DBType
		if opt.DBType == Sharded {
			retDB.Spec.ShardTopology = &kdm.MongoDBShardingTopology{
				Shard: kdm.MongoDBShardNode{
					MongoDBNode: kdm.MongoDBNode{
						Replicas:    3,
						PodTemplate: podTmpl,
					},
					Shards:  2,
					Storage: storage,
				},
				ConfigServer: kdm.MongoDBConfigNode{
					MongoDBNode: kdm.MongoDBNode{
						Replicas:    3,
						PodTemplate: podTmpl,
					},
					Storage: storage,
				},
				Mongos: kdm.MongoDBMongosNode{
					MongoDBNode: kdm.MongoDBNode{
						Replicas:    2,
						PodTemplate: podTmpl,
					},
				},
			}
		} else if opt.DBType == ReplicaSet {
			retDB.Spec.ReplicaSet = &kdm.MongoDBReplicaSet{
				Name: "rs",
			}
			retDB.Spec.PodTemplate = &podTmpl
			retDB.Spec.Replicas = func(i int32) *int32 { return &i }(3)
			retDB.Spec.StorageType = kdm.StorageTypeDurable
			retDB.Spec.Storage = storage
		} else if opt.DBType == StandAlone {
			retDB.Spec.StorageType = kdm.StorageTypeDurable
			retDB.Spec.Storage = storage
		}
		// TLS
		if opt.SslModeEnabled {
			retDB.Spec.SSLMode = kdm.SSLModeRequireSSL
			retDB.Spec.TLS = &clientutil.TLSConfig{
				IssuerRef: &core.TypedLocalObjectReference{
					APIGroup: func(s string) *string { return &s }("cert-manager.io"),
					Kind:     "Issuer",
					Name:     "ca-issuer",
				},
			}
			retDB.Spec.ClusterAuthMode = kdm.ClusterAuthModeX509
		}
	}
	return retDB
}

func (i *TestOptions) CreateMongoDB() error {
	err := i.myClient.Create(context.TODO(), i.Mongodb)
	return err
}

func (i *TestOptions) DeleteMongoDB() error {
	err := i.myClient.Delete(context.TODO(), i.Mongodb)
	return err
}
