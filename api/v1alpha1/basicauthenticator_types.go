/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BasicAuthenticatorSpec defines the desired state of BasicAuthenticator
type BasicAuthenticatorSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=sidecar;deployment
	// Type is used to determine that nginx should be sidercar or deployment
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:validation:Minimum=1
	Replicas int `json:"replicas"`

	// +kubebuilder:validation:Optional
	Selector metav1.LabelSelector `json:"selector"`

	// +kubebuilder:validation:Optional
	AppPort int `json:"appPort"`

	// +kubebuilder:validation:Optional
	AppService string `json:"appService"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AdaptiveScale bool `json:"adaptiveScale"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=80
	AuthenticatorPort int `json:"authenticatorPort"`

	// +kubebuilder:validation:Optional
	CredentialsSecretRef string `json:"credentialsSecretRef"`
}

// BasicAuthenticatorStatus defines the observed state of BasicAuthenticator
type BasicAuthenticatorStatus struct {
	ReadyReplicas int    `json:"readyReplicas"`
	Reason        string `json:"reason"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BasicAuthenticator is the Schema for the basicauthenticators API
type BasicAuthenticator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BasicAuthenticatorSpec   `json:"spec,omitempty"`
	Status BasicAuthenticatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BasicAuthenticatorList contains a list of BasicAuthenticator
type BasicAuthenticatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BasicAuthenticator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BasicAuthenticator{}, &BasicAuthenticatorList{})
}
