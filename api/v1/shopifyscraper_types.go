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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ShopifyScraperSpec defines the desired state of ShopifyScraper.
type ShopifyScraperSpec struct {
	Name      string `json:"name"`
	Url       string `json:"url"`
	WatchTime *int32 `json:"watchtime"`
}

// ShopifyScraperStatus defines the observed state of ShopifyScraper.
type ShopifyScraperStatus struct {
	Active []corev1.ObjectReference `json:"active,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ShopifyScraper is the Schema for the shopifyscrapers API.
type ShopifyScraper struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShopifyScraperSpec   `json:"spec,omitempty"`
	Status ShopifyScraperStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ShopifyScraperList contains a list of ShopifyScraper.
type ShopifyScraperList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShopifyScraper `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShopifyScraper{}, &ShopifyScraperList{})
}
