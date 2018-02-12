/*
Copyright 2017 The Kubernetes Authors.

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
	appsv1 "k8s.io/api/apps/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataStore is a specification for a DataStore resource
type DataStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataStoreSpec   `json:"spec"`
	Status DataStoreStatus 		`json:"status"`
}

// DatastoreSpec is the spec for a Datastore resource
type DataStoreSpec struct {
	appsv1.StatefulSetSpec
	DeploymentName string `json:"deploymentName"`
	Replicas       *int32 `json:"replicas"`
}

// DataStoreStatus is the status for a Datastore resource
type DataStoreStatus struct {
	Replicas 				*int32							`json:"availableReplicas"`
	ReadyReplicas 			*int32							`json:"readyReplicas"`
	ScalingDown				bool							`json:"scalingDown"`
	RestartScaling			bool							`json:"restartScaling"`
	InitialReplicas			*int32							`json:"initialReplicas"`
	FinalReplicas			*int32							`json:"finalReplicas"`
	PodTerminationStatuses	[]DataStorePodTerminationStatus	`json:"podTerminationStatuses"`
}

func (s *DataStoreStatus) SetPodTerminationStatus(name string, decommissioned bool) {
	for _, status := range s.PodTerminationStatuses {
		if status.PodName == name {
			status.Decommissioned = decommissioned
			return;
		}
	}


	edited := append(s.PodTerminationStatuses, DataStorePodTerminationStatus{
		PodName:name,
		Decommissioned:decommissioned,
	})
	s.PodTerminationStatuses = edited
}

func (s *DataStoreStatus) GetPodTerminationStatus(name string) *DataStorePodTerminationStatus {
	for _, status := range s.PodTerminationStatuses {
		if status.PodName == name {
			return &status
		}
	}
	return nil
}

func (s *DataStoreStatus) RemovePodTerminationStatus(name string) {
	index := -1
	for i, status := range s.PodTerminationStatuses {
		if status.PodName == name {
			index = i
			break
		}
	}

	if index >= 0 {
		edited := append(s.PodTerminationStatuses[:index], s.PodTerminationStatuses[index + 1:]...)
		s.PodTerminationStatuses = edited
	}
}

type DataStorePodTerminationStatus struct {
	PodName			string
	Decommissioned	bool
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatastoreList is a list of Datastore resources
type DataStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []DataStore `json:"items"`
}
