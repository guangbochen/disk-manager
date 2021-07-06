/*
Copyright 2021 Rancher Labs, Inc.

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

// Code generated by main. DO NOT EDIT.

// +k8s:deepcopy-gen=package
// +groupName=longhorn.io
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlockDeviceList is a list of BlockDevice resources
type BlockDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BlockDevice `json:"items"`
}

func NewBlockDevice(namespace, name string, obj BlockDevice) *BlockDevice {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("BlockDevice").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeList is a list of Node resources
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Node `json:"items"`
}

func NewNode(namespace, name string, obj Node) *Node {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("Node").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeList is a list of Volume resources
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Volume `json:"items"`
}

func NewVolume(namespace, name string, obj Volume) *Volume {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("Volume").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}
