/*
Copyright 2024 The Kairos CAPI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.
*/

package v1beta2

// Condition types for KairosControlPlane
const (
	// AvailableCondition indicates that the control plane is available
	AvailableCondition = "Available"
)

// Condition reasons
const (
	// WaitingForMachinesReason indicates that the control plane is waiting for machines
	WaitingForMachinesReason = "WaitingForMachines"

	// WaitingForMachinesReadyReason indicates that the control plane is waiting for machines to be ready
	WaitingForMachinesReadyReason = "WaitingForMachinesReady"

	// ControlPlaneInitializationFailedReason indicates that control plane initialization failed
	ControlPlaneInitializationFailedReason = "ControlPlaneInitializationFailed"

	// ControlPlaneInitializationSucceededReason indicates that control plane initialization succeeded
	ControlPlaneInitializationSucceededReason = "ControlPlaneInitializationSucceeded"

	// ScalingUpReason indicates that the control plane is scaling up
	ScalingUpReason = "ScalingUp"

	// ScalingDownReason indicates that the control plane is scaling down
	ScalingDownReason = "ScalingDown"
)

