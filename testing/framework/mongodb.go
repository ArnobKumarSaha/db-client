package framework

import (
	"context"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
)

func (i *Invocation) GetMongoShardSpec() *kdm.MongoDB {
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
	}

	return &kdm.MongoDB{
		ObjectMeta: meta.ObjectMeta{
			Name:      "mng-shrd",
			Namespace: i.Namespace(),
		},
		Spec: kdm.MongoDBSpec{
			Version: "4.4.6",
			ShardTopology: &kdm.MongoDBShardingTopology{
				Shard: kdm.MongoDBShardNode{
					MongoDBNode: kdm.MongoDBNode{
						Replicas:    2,
						PodTemplate: podTmpl,
					},
					Shards:  1,
					Storage: storage,
				},
				ConfigServer: kdm.MongoDBConfigNode{
					MongoDBNode: kdm.MongoDBNode{
						Replicas:    1,
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
			},
		},
	}
}

func (i *Invocation) CreateMongoDB(m *kdm.MongoDB) error {
	err := i.myClient.Create(context.TODO(), m)
	return err
}
func (i *Invocation) DeleteMongoDB(m *kdm.MongoDB) error {
	err := i.myClient.Delete(context.TODO(), m)
	return err
}