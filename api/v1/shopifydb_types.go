/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ShopifyDBSpec defines the desired state of ShopifyDB.
type ShopifyDBSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ShopifyDB. Edit shopifydb_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ShopifyDBStatus defines the observed state of ShopifyDB.
type ShopifyDBStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ShopifyDB is the Schema for the shopifydbs API.
type ShopifyDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShopifyDBSpec   `json:"spec,omitempty"`
	Status ShopifyDBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ShopifyDBList contains a list of ShopifyDB.
type ShopifyDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShopifyDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShopifyDB{}, &ShopifyDBList{})
}
