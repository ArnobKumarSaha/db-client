package v1

import (
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MyDatabaseSpec struct {
	// A database with name "DataBaseName" will be created in the primary replica
	DatabaseName string `json:"database_name,omitempty"`

	// a collection with name "CollectionName" will also be created
	CollectionName string `json:"collection_name,omitempty"`
}

type MyDatabaseStatus struct {

}

// MongoDBOptions represents options that can be used to configure a Database.
type MongoDBOptions struct {
	// The read concern to use for operations executed on the Database. The default value is nil, which means that
	// the read concern of the client used to configure the Database will be used.
	ReadConcern *readconcern.ReadConcern

	// The write concern to use for operations executed on the Database. The default value is nil, which means that the
	// write concern of the client used to configure the Database will be used.
	WriteConcern *writeconcern.WriteConcern

	// The read preference to use for operations executed on the Database. The default value is nil, which means that
	// the read preference of the client used to configure the Database will be used.
	ReadPreference *readpref.ReadPref

	// The BSON registry to marshal and unmarshal documents for operations executed on the Database. The default value
	// is nil, which means that the registry of the client used to configure the Database will be used.
	Registry *bsoncodec.Registry
}


//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type MyDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyDatabaseSpec   `json:"spec,omitempty"`
	Status MyDatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type MyDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyDatabase{}, &MyDatabaseList{})
}