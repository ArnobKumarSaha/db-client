/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "kmodules.xyz/client-go/api/v1"
	v1 "kmodules.xyz/offshoot-api/api/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MongoDBDatabaseSpec defines the desired state of MongoDBDatabase
type MongoDBDatabaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The reference to the Database of kind apimachinery/apis/kubedb
	DatabaseRef DatabaseRef `json:"databaseRef,omitempty"`

	// Reference to the VaultServer
	VaultRef VaultRef `json:"vaultRef,omitempty"`

	// To assing an actual database instance to an user
	DatabaseSchema DatabaseSchema `json:"databaseSchema,omitempty"`

	// The list of ServiceAccounts those will have some certain roles
	Subjects []Subject `json:"subjects,omitempty"`

	// Init is used to initialize database
	// +optional
	Init *InitSpec `json:"init,omitempty"`

	// For restore using stash
	// +optional
	Restore *RestoreRef `json:"restore,omitempty"`
	//Restore *stash.RestoreSessionSpec `json:"restore,omitempty"`

	// DeletionPolicy controls the delete operation for database
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

type RestoreRef struct {
	Repository apiv1.ObjectReference `json:"repository,omitempty"`
	Snapshot   string                `json:"snapshot,omitempty"`
}

type DatabaseRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type VaultRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type DatabaseSchema struct {
	Name string `json:"name"`
}

type Subject struct {
	SubjectKind metav1.TypeMeta `json:"kind"`
	Name        string          `json:"name"`
	Namespace   string          `json:"namespace"`
}

type InitSpec struct {
	// Initialized indicates that this database has been initialized.
	// This will be set by the operator when status.conditions["Provisioned"] is set to ensure
	// that database is not mistakenly reset when recovered using disaster recovery tools.
	Initialized bool `json:"initialized,omitempty" protobuf:"varint,1,opt,name=initialized"`
	// Wait for initial DataRestore condition
	WaitForInitialRestore bool              `json:"waitForInitialRestore,omitempty" protobuf:"varint,2,opt,name=waitForInitialRestore"`
	Script                *ScriptSourceSpec `json:"script,omitempty" protobuf:"bytes,1,opt,name=scriptPath"`

	// This will take some database related config from the user
	PodTemplate *v1.PodTemplateSpec `json:"podTemplate,omitempty"`
}

type ScriptSourceSpec struct {
	ScriptPath   string            `json:"scriptPath,omitempty" protobuf:"bytes,1,opt,name=scriptPath"`
	VolumeSource core.VolumeSource `json:"volumeSource,omitempty" protobuf:"bytes,2,opt,name=volumeSource"`
}

// MongoDBDatabaseStatus defines the observed state of MongoDBDatabase
type MongoDBDatabaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MongoDBDatabase is the Schema for the mongodbdatabases API
type MongoDBDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBDatabaseSpec   `json:"spec,omitempty"`
	Status MongoDBDatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MongoDBDatabaseList contains a list of MongoDBDatabase
type MongoDBDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoDBDatabase{}, &MongoDBDatabaseList{})
}
