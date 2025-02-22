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

// DiscordBotSpec defines the desired state of DiscordBot.
type DiscordBotSpec struct {
	Token string `json:"key,omitempty"` //omitempty for tests
}

// DiscordBotStatus defines the observed state of DiscordBot.
type DiscordBotStatus struct {
	Running bool `json:"running"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DiscordBot is the Schema for the discordbots API.
type DiscordBot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscordBotSpec   `json:"spec,omitempty"`
	Status DiscordBotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiscordBotList contains a list of DiscordBot.
type DiscordBotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscordBot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DiscordBot{}, &DiscordBotList{})
}
