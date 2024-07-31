/*
Copyright 2022 The Crossplane Authors.

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
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CtrlDriftParameters are the configurable fields of a CtrlDrift.
type CtrlDriftParameters struct {
	DeployName      string `json:"deploy_name"`
	DeployNamespace string `json:"deploy_namespace"`
	TrainingScript  string `json:"training_script"`
}

// CtrlDriftObservation are the observable fields of a CtrlDrift.
type CtrlDriftObservation struct {
	Drift string `json:"drift"`
}

// A CtrlDriftSpec defines the desired state of a CtrlDrift.
type CtrlDriftSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       CtrlDriftParameters `json:"forProvider"`
}

// A CtrlDriftStatus represents the observed state of a CtrlDrift.
type CtrlDriftStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          CtrlDriftObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A CtrlDrift is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,driftprovider}
type CtrlDrift struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CtrlDriftSpec   `json:"spec"`
	Status CtrlDriftStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CtrlDriftList contains a list of CtrlDrift
type CtrlDriftList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CtrlDrift `json:"items"`
}

// CtrlDrift type metadata.
var (
	CtrlDriftKind             = reflect.TypeOf(CtrlDrift{}).Name()
	CtrlDriftGroupKind        = schema.GroupKind{Group: Group, Kind: CtrlDriftKind}.String()
	CtrlDriftKindAPIVersion   = CtrlDriftKind + "." + SchemeGroupVersion.String()
	CtrlDriftGroupVersionKind = SchemeGroupVersion.WithKind(CtrlDriftKind)
)

func init() {
	SchemeBuilder.Register(&CtrlDrift{}, &CtrlDriftList{})
}
