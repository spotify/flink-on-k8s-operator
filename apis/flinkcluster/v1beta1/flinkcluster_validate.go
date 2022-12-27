/*
Copyright 2019 Google LLC.

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

package v1beta1

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	InvalidControlAnnMsg           = "invalid value for annotation key: %v, value: %v, available values: savepoint, job-cancel"
	InvalidJobStateForJobCancelMsg = "job-cancel is not allowed because job is not started yet or already terminated, annotation: %v"
	InvalidJobStateForSavepointMsg = "savepoint is not allowed because job is not started yet or already stopped, annotation: %v"
	InvalidSavepointDirMsg         = "savepoint is not allowed without spec.job.savepointsDir, annotation: %v"
	SessionClusterWarnMsg          = "%v is not allowed for session cluster, annotation: %v"
	ControlChangeWarnMsg           = "change is not allowed for control in progress, annotation: %v"
	dns1035ErrorMsg                = "cluster name %s is invalid: a DNS-1035 name must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name', or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?'"
	maxClusterNameLength           = 49 // 63 - 14 (max suffix length)
)

// Validator validates CUD requests for the CR.
type Validator struct{}

// ValidateCreate validates create request.
func (v *Validator) ValidateCreate(cluster *FlinkCluster) error {
	var err error
	err = v.validateMeta(&cluster.ObjectMeta)
	if err != nil {
		return err
	}

	var flinkVersion *version.Version
	if len(cluster.Spec.FlinkVersion) != 0 {
		flinkVersion, err = version.NewVersion(cluster.Spec.FlinkVersion)
		if err != nil {
			return err
		}
	}

	err = v.validateGCPConfig(cluster.Spec.GCPConfig)
	if err != nil {
		return err
	}
	err = v.validateJobManager(flinkVersion, cluster.Spec.JobManager)
	if err != nil {
		return err
	}
	err = v.validateTaskManager(flinkVersion, cluster.Spec.TaskManager)
	if err != nil {
		return err
	}
	err = v.validateJob(cluster.Spec.Job)
	if err != nil {
		return err
	}
	return nil
}

// ValidateUpdate validates update request.
func (v *Validator) ValidateUpdate(old *FlinkCluster, new *FlinkCluster) error {
	var err error
	controlName, err := v.checkControlAnnotations(old, new)
	if err != nil {
		return err
	}

	// Skip remaining validation if no changes in spec.
	if reflect.DeepEqual(new.Spec, old.Spec) {
		return nil
	}

	cancelRequested, err := v.checkCancelRequested(old, new)
	if err != nil {
		return err
	}
	if cancelRequested || controlName == ControlNameJobCancel {
		return nil
	}

	savepointGenUpdated, err := v.checkSavepointGeneration(old, new)
	if err != nil {
		return err
	}
	if savepointGenUpdated {
		return nil
	}

	err = v.validateTaskManagerUpdate(old, new)
	if err != nil {
		return err
	}

	err = v.validateJobUpdate(old, new)
	if err != nil {
		return err
	}

	err = v.ValidateCreate(new)
	if err != nil {
		return err
	}

	return nil
}

func (v *Validator) checkControlAnnotations(old *FlinkCluster, new *FlinkCluster) (string, error) {
	oldUserControl := old.Annotations[ControlAnnotation]
	newUserControl, ok := new.Annotations[ControlAnnotation]
	if ok {
		if oldUserControl != newUserControl && old.Status.Control != nil && old.Status.Control.State == ControlStateInProgress {
			return "", fmt.Errorf(ControlChangeWarnMsg, ControlAnnotation)
		}
		switch newUserControl {
		case ControlNameJobCancel:
			var job = old.Status.Components.Job
			if old.Spec.Job == nil {
				return "", fmt.Errorf(SessionClusterWarnMsg, ControlNameJobCancel, ControlAnnotation)
			} else if job == nil || job.IsTerminated(old.Spec.Job) {
				return "", errors.NewResourceExpired(fmt.Sprintf(InvalidJobStateForJobCancelMsg, ControlAnnotation))
			} else {
				return ControlNameJobCancel, nil
			}
		case ControlNameSavepoint:
			var job = old.Status.Components.Job
			if old.Spec.Job == nil {
				return "", fmt.Errorf(SessionClusterWarnMsg, ControlNameSavepoint, ControlAnnotation)
			} else if old.Spec.Job.SavepointsDir == nil || *old.Spec.Job.SavepointsDir == "" {
				return "", fmt.Errorf(InvalidSavepointDirMsg, ControlAnnotation)
			} else if job == nil || job.IsStopped() {
				return "", fmt.Errorf(InvalidJobStateForSavepointMsg, ControlAnnotation)
			} else {
				return ControlNameSavepoint, nil
			}
		default:
			return "", fmt.Errorf(InvalidControlAnnMsg, ControlAnnotation, newUserControl)
		}
	}
	return "", nil
}

func (v *Validator) checkCancelRequested(
	old *FlinkCluster, new *FlinkCluster) (bool, error) {
	if old.Spec.Job == nil || new.Spec.Job == nil {
		return false, nil
	}
	var restartJob = (old.Spec.Job.CancelRequested != nil && *old.Spec.Job.CancelRequested) &&
		(new.Spec.Job.CancelRequested == nil || !*new.Spec.Job.CancelRequested)
	if restartJob {
		return false, fmt.Errorf(
			"updating cancelRequested from true to false is not allowed")
	}

	var stopJob = (old.Spec.Job.CancelRequested == nil || !*old.Spec.Job.CancelRequested) &&
		(new.Spec.Job.CancelRequested != nil && *new.Spec.Job.CancelRequested)
	if stopJob {
		// Check if only `cancelRequested` changed, no other changes.
		var oldCopy = old.DeepCopy()
		oldCopy.Spec.Job.CancelRequested = new.Spec.Job.CancelRequested

		if reflect.DeepEqual(new.Spec, oldCopy.Spec) {
			return true, nil
		}

		return false, fmt.Errorf(
			"you cannot update cancelRequested with others at the same time")
	}

	return false, nil
}

func (v *Validator) checkSavepointGeneration(
	old *FlinkCluster, new *FlinkCluster) (bool, error) {
	if old.Spec.Job == nil || new.Spec.Job == nil {
		return false, nil
	}

	var oldSpecGen = old.Spec.Job.SavepointGeneration
	var newSpecGen = new.Spec.Job.SavepointGeneration
	if oldSpecGen == newSpecGen {
		return false, nil
	}

	var oldStatusGen int32 = 0
	if old.Status.Components.Job != nil {
		oldStatusGen = old.Status.Components.Job.SavepointGeneration
	}
	if newSpecGen != oldStatusGen+1 {
		return false, fmt.Errorf(
			"you can only update savepointGeneration to %v",
			oldStatusGen+1)
	}

	// Check if only `savepointGeneration` changed, no other changes.
	var oldCopy = old.DeepCopy()
	oldCopy.Spec.Job.SavepointGeneration = newSpecGen
	if reflect.DeepEqual(new.Spec, oldCopy.Spec) {
		return true, nil
	}

	return false, fmt.Errorf(
		"you cannot update savepointGeneration with others at the same time")
}

func (v *Validator) validateTaskManagerUpdate(old *FlinkCluster, new *FlinkCluster) error {
	//  When flink-operator updated, old flinkCluster does not have TaskManager.DeploymentType, may cause update failed.
	oldDeploymentType := old.Spec.TaskManager.DeploymentType
	newDeploymentType := new.Spec.TaskManager.DeploymentType

	if oldDeploymentType == "" && (newDeploymentType != "" && newDeploymentType != DeploymentTypeStatefulSet) {
		return fmt.Errorf(
			"updating deploymentType is not allowed")
	}
	if oldDeploymentType != "" && (oldDeploymentType != newDeploymentType) {
		return fmt.Errorf(
			"updating deploymentType is not allowed")
	}
	return nil
}

// Validate job update.
func (v *Validator) validateJobUpdate(old *FlinkCluster, new *FlinkCluster) error {
	switch {
	case old.Spec.Job == nil && new.Spec.Job == nil:
		return nil
	case old.Spec.Job == nil || new.Spec.Job == nil:
		oldJobSpec, _ := json.Marshal(old.Spec.Job)
		newJobSpec, _ := json.Marshal(new.Spec.Job)
		return fmt.Errorf("you cannot change cluster type between session cluster and job cluster, old spec.job: %q, new spec.job: %q", oldJobSpec, newJobSpec)
	case old.Spec.Job.SavepointsDir == nil || *old.Spec.Job.SavepointsDir == "":
		return fmt.Errorf("updating job is not allowed when spec.job.savepointsDir was not provided")
	case old.Spec.Job.SavepointsDir != nil && *old.Spec.Job.SavepointsDir != "" &&
		(new.Spec.Job.SavepointsDir == nil || *new.Spec.Job.SavepointsDir == ""):
		return fmt.Errorf("removing savepointsDir is not allowed")
	case old.IsHighAvailabilityEnabled() != new.IsHighAvailabilityEnabled():
		return fmt.Errorf("updating highAvailability settings is not allowed")
	case !isBlank(new.Spec.Job.FromSavepoint):
		return nil
	default:
		// In the case of taking savepoint is skipped, check if the savepoint is up-to-date.
		var oldJob = old.Status.Components.Job
		var takeSavepointOnUpdate = new.Spec.Job.TakeSavepointOnUpdate == nil || *new.Spec.Job.TakeSavepointOnUpdate
		var skipTakeSavepoint = !takeSavepointOnUpdate || oldJob.IsStopped()
		var now = time.Now()
		if skipTakeSavepoint && oldJob != nil && !oldJob.UpdateReady(new.Spec.Job, now) {
			oldJobJson, _ := json.Marshal(oldJob)
			var takeSP, maxStateAge string
			if new.Spec.Job.TakeSavepointOnUpdate == nil {
				takeSP = "nil"
			} else {
				takeSP = strconv.FormatBool(*new.Spec.Job.TakeSavepointOnUpdate)
			}
			if new.Spec.Job.MaxStateAgeToRestoreSeconds == nil {
				maxStateAge = "nil"
			} else {
				maxStateAge = strconv.Itoa(int(*new.Spec.Job.MaxStateAgeToRestoreSeconds))
			}
			return fmt.Errorf("cannot update spec: taking savepoint is skipped but no up-to-date savepoint, "+
				"spec.job.takeSavepointOnUpdate: %v, spec.job.maxStateAgeToRestoreSeconds: %v, job status: %q",
				takeSP, maxStateAge, oldJobJson)
		}
	}
	return nil
}

func (v *Validator) validateMeta(meta *metav1.ObjectMeta) error {
	if len(meta.Name) == 0 || len(meta.Name) >= maxClusterNameLength {
		return fmt.Errorf("cluster name size needs to greater than 0 and less than 50")
	}
	// cluster name is used as the prefix of almost all resources, so it must be a valid DNS label.
	if len(validation.NameIsDNS1035Label(meta.Name, false)) > 0 {
		return fmt.Errorf(dns1035ErrorMsg, meta.Name)
	}

	if len(meta.Namespace) == 0 {
		return fmt.Errorf("cluster namesapce is unspecified")
	}
	nsErrors := validation.ValidateNamespaceName(meta.Namespace, false)
	if len(nsErrors) > 0 {
		err := strings.Join(nsErrors, ",")
		return fmt.Errorf("cluster namespace %s is invalid: %s", meta.Namespace, err)
	}

	return nil
}

func (v *Validator) validateGCPConfig(gcpConfig *GCPConfig) error {
	if gcpConfig == nil {
		return nil
	}
	var saConfig = gcpConfig.ServiceAccount
	if saConfig != nil {
		if len(saConfig.SecretName) == 0 {
			return fmt.Errorf("GCP service account secret name is unspecified")
		}
		if len(saConfig.KeyFile) == 0 {
			return fmt.Errorf("GCP service account key file name is unspecified")
		}
		if len(saConfig.MountPath) == 0 {
			return fmt.Errorf("GCP service account volume mount path is unspecified")
		}
		if strings.HasSuffix(saConfig.MountPath, saConfig.KeyFile) {
			return fmt.Errorf("invalid GCP service account volume mount path")
		}
	}
	return nil
}

func (v *Validator) validateJobManager(flinkVersion *version.Version, jmSpec *JobManagerSpec) error {
	var err error
	if jmSpec == nil {
		return nil
	}

	fp := field.NewPath("spec.jobManager")
	if errors := validation.ValidateAnnotations(jmSpec.PodAnnotations, fp.Child("podAnnotations")); len(errors) > 0 {
		return fmt.Errorf(errors.ToAggregate().Error())
	}
	if errors := v1validation.ValidateLabels(jmSpec.PodLabels, fp.Child("podLabels")); len(errors) > 0 {
		return fmt.Errorf(errors.ToAggregate().Error())
	}

	// Ports.
	var ports = []NamedPort{
		{Name: "rpc", ContainerPort: *jmSpec.Ports.RPC},
		{Name: "blob", ContainerPort: *jmSpec.Ports.Blob},
		{Name: "query", ContainerPort: *jmSpec.Ports.Query},
		{Name: "ui", ContainerPort: *jmSpec.Ports.UI},
	}
	ports = append(ports, jmSpec.ExtraPorts...)
	err = v.checkDupPorts(ports, "jobmanager")
	if err != nil {
		return err
	}

	if err := v.validateResourceRequirements(jmSpec.Resources, "jobmanager"); err != nil {
		return err
	}

	if flinkVersion == nil || flinkVersion.LessThan(v10) {
		if jmSpec.MemoryProcessRatio != nil {
			return fmt.Errorf("MemoryProcessRatio config cannot be used with flinkVersion < 1.11', use " +
				"memoryOffHeapRatio instead")
		}

		// MemoryOffHeapRatio
		err = v.validateRatio(jmSpec.MemoryOffHeapRatio, "jobmanager", "memoryOffHeapRatio")
		if err != nil {
			return err
		}

		// MemoryOffHeapMin
		err = v.validateMemoryOffHeapMin(&jmSpec.MemoryOffHeapMin, jmSpec.GetResources().Memory(), "jobmanager")
		if err != nil {
			return err
		}
	} else {
		if jmSpec.MemoryOffHeapRatio != nil || !jmSpec.MemoryOffHeapMin.IsZero() {
			return fmt.Errorf("MemoryOffHeapRatio or MemoryOffHeapMin config cannot be used with flinkVersion >= 1.11'; " +
				"use memoryProcessRatio istead")
		}
		// MemoryProcessRatio
		err = v.validateRatio(jmSpec.MemoryProcessRatio, "jobmanager", "memoryProcessRatio")
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validateTaskManager(flinkVersion *version.Version, tmSpec *TaskManagerSpec) error {
	if tmSpec == nil {
		return nil
	}

	fp := field.NewPath("spec.taskManager")
	if errors := validation.ValidateAnnotations(tmSpec.PodAnnotations, fp.Child("podAnnotations")); len(errors) > 0 {
		return fmt.Errorf(errors.ToAggregate().Error())
	}
	if errors := v1validation.ValidateLabels(tmSpec.PodLabels, fp.Child("podLabels")); len(errors) > 0 {
		return fmt.Errorf(errors.ToAggregate().Error())
	}

	// Ports.
	var err error
	var ports = []NamedPort{
		{Name: "rpc", ContainerPort: *tmSpec.Ports.RPC},
		{Name: "data", ContainerPort: *tmSpec.Ports.Data},
		{Name: "query", ContainerPort: *tmSpec.Ports.Query},
	}
	ports = append(ports, tmSpec.ExtraPorts...)
	err = v.checkDupPorts(ports, "taskmanager")
	if err != nil {
		return err
	}

	if err := v.validateResourceRequirements(tmSpec.Resources, "taskmanager"); err != nil {
		return err
	}

	if flinkVersion == nil || flinkVersion.LessThan(v10) {
		if tmSpec.MemoryProcessRatio != nil {
			return fmt.Errorf("MemoryProcessRatio config cannot be used with flinkVersion < 1.11', use " +
				"memoryOffHeapRatio instead")
		}

		// MemoryOffHeapRatio
		err = v.validateRatio(tmSpec.MemoryOffHeapRatio, "taskmanager", "memoryOffHeapRatio")
		if err != nil {
			return err
		}

		// MemoryOffHeapMin
		err = v.validateMemoryOffHeapMin(&tmSpec.MemoryOffHeapMin, tmSpec.GetResources().Memory(), "taskmanager")
		if err != nil {
			return err
		}
	} else {
		if tmSpec.MemoryOffHeapRatio != nil || !tmSpec.MemoryOffHeapMin.IsZero() {
			return fmt.Errorf("MemoryOffHeapRatio or MemoryOffHeapMin config cannot be used with flinkVersion >= 1.11'; " +
				"use memoryProcessRatio istead")
		}
		// MemoryProcessRatio
		err = v.validateRatio(tmSpec.MemoryProcessRatio, "taskmanager", "memoryProcessRatio")
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validateJob(jobSpec *JobSpec) error {
	if jobSpec == nil {
		return nil
	}

	fp := field.NewPath("spec.job")
	if errors := validation.ValidateAnnotations(jobSpec.PodAnnotations, fp.Child("podAnnotations")); len(errors) > 0 {
		return fmt.Errorf(errors.ToAggregate().Error())
	}
	if errors := v1validation.ValidateLabels(jobSpec.PodLabels, fp.Child("podLabels")); len(errors) > 0 {
		return fmt.Errorf(errors.ToAggregate().Error())
	}

	applicationMode := jobSpec.Mode != nil && *jobSpec.Mode == JobModeApplication
	if !applicationMode && jobSpec.JarFile == nil && jobSpec.PyFile == nil && jobSpec.PyModule == nil {
		return fmt.Errorf("job jarFile or pythonFile or pythonModule is unspecified")
	}

	if jobSpec.Parallelism != nil && *jobSpec.Parallelism < 1 {
		return fmt.Errorf("job parallelism must be >= 1")
	}

	switch *jobSpec.RestartPolicy {
	case JobRestartPolicyNever:
	case JobRestartPolicyFromSavepointOnFailure:
		if jobSpec.MaxStateAgeToRestoreSeconds == nil {
			return fmt.Errorf("maxStateAgeToRestoreSeconds must be specified when restartPolicy is set as FromSavepointOnFailure")
		}
	default:
		return fmt.Errorf("invalid job restartPolicy: %v", *jobSpec.RestartPolicy)
	}

	if jobSpec.TakeSavepointOnUpdate != nil && !*jobSpec.TakeSavepointOnUpdate &&
		jobSpec.MaxStateAgeToRestoreSeconds == nil {
		return fmt.Errorf("maxStateAgeToRestoreSeconds must be specified when takeSavepointOnUpdate is set as false")
	}

	if jobSpec.CancelRequested != nil && *jobSpec.CancelRequested {
		return fmt.Errorf(
			"property `cancelRequested` cannot be set to true for a new job")
	}

	return nil
}

func (v *Validator) validateResourceRequirements(rr corev1.ResourceRequirements, component string) error {
	memoryNotSet := true
	cpuNotSet := true

	if rr.Requests == nil && rr.Limits == nil {
		return fmt.Errorf("%s resource requests/limits are unspecified", component)
	}

	if rr.Requests != nil {
		cpuNotSet = cpuNotSet && rr.Requests.Cpu().IsZero()
		memoryNotSet = memoryNotSet && rr.Requests.Memory().IsZero()
	}

	if rr.Limits != nil {
		cpuNotSet = cpuNotSet && rr.Limits.Cpu().IsZero()
		memoryNotSet = memoryNotSet && rr.Limits.Memory().IsZero()
	}

	if cpuNotSet {
		return fmt.Errorf("%s cpu request/limit is unspecified", component)
	}

	if memoryNotSet {
		return fmt.Errorf("%s memory request/limit is unspecified", component)
	}

	return nil
}

// Check duplicate name and number in NamedPort array.
func (v *Validator) checkDupPorts(ports []NamedPort, component string) error {
	if len(ports) == 0 {
		return nil
	}
	var portNameSet = make(map[string]bool)
	var portNumberSet = make(map[int32]bool)
	for _, port := range ports {
		if portNumberSet[port.ContainerPort] {
			return fmt.Errorf("duplicate containerPort %v in %v, each port number of ports and extraPorts must be unique", port.ContainerPort, component)
		}
		portNumberSet[port.ContainerPort] = true
		if port.Name != "" {
			if portNameSet[port.Name] {
				return fmt.Errorf("duplicate port name %v in %v, each port name of ports and extraPorts must be unique", port.Name, component)
			}
			portNameSet[port.Name] = true
		}
	}
	return nil
}

func (v *Validator) validateCleanupAction(
	property string, value CleanupAction) error {
	switch value {
	case CleanupActionDeleteCluster:
	case CleanupActionDeleteTaskManager:
	case CleanupActionKeepCluster:
	default:
		return fmt.Errorf(
			"invalid %v: %v",
			property, value)
	}
	return nil
}

func (v *Validator) validateJobMode(property string, value JobMode) error {
	switch value {
	case JobModeBlocking:
	case JobModeApplication:
	case JobModeDetached:
	default:
		return fmt.Errorf("invalid %v: %v", property, value)
	}
	return nil
}

func (v *Validator) validateRatio(ratio *int32, component, property string) error {
	if ratio == nil || *ratio > 100 || *ratio < 0 {
		return fmt.Errorf("invalid %v %v, it must be between 0 and 100", component, property)
	}
	return nil
}

func (v *Validator) validateMemoryOffHeapMin(
	offHeapMin *resource.Quantity, memoryLimit *resource.Quantity, component string) error {
	if offHeapMin == nil {
		return fmt.Errorf("invalid %v memory configuration, MemoryOffHeapMin is not specified", component)
	} else if memoryLimit.Value() > 0 {
		if offHeapMin.Value() > memoryLimit.Value() {
			return fmt.Errorf("invalid %v memory configuration, memory limit must be larger than MemoryOffHeapMin, "+
				"memory limit: %d bytes, memoryOffHeapMin: %d bytes", component, memoryLimit.Value(), offHeapMin.Value())
		}
	}
	return nil
}
