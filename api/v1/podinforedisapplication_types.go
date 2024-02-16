/*
Copyright 2024.

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
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// PodInfoRedisApplicationSpec defines the desired state of PodInfoRedisApplication
type PodInfoRedisApplicationSpec struct {

	// Replica count of PodInfo.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum:=1
	// +optional
	ReplicaCount *int32 `json:"replicaCount,omitempty"`
	Resources    `json:"resources,omitempty"`
	Image        `json:"image,omitempty"`
	UI           `json:"ui,omitempty"`
	Redis        `json:"redis,omitempty"`
}

type Resources struct {
	MemoryLimit resource.Quantity `json:"memoryLimit,omitempty"`
	CpuRequest  resource.Quantity `json:"cpuRequest,omitempty"`
}

type Image struct {
	// Repository of the PodInfo conatiner image.
	Repository string `json:"repository,omitempty"`
	// Tag of the PodInfo container image.
	Tag string `json:"tag,omitempty"`
}

type UI struct {
	// Hexadecimal color string of the PodInfo UI.
	Color string `json:"color,omitempty"`
	// PodInfo message to display.
	Message string `json:"message,omitempty"`
}

type Redis struct {
	// Enables a Redis datastore for PodInfo containers.
	Enabled bool `json:"enabled,omitempty"`
}

// PodInfoRedisApplicationStatus defines the observed state of PodInfoRedisApplication
type PodInfoRedisApplicationStatus struct {
	// TODO add Conditions
}

// PodInfoRedisApplication is the Schema for the podinforedisapplication API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=podinforedisapplication,shortName={pira}
type PodInfoRedisApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodInfoRedisApplicationSpec   `json:"spec,omitempty"`
	Status PodInfoRedisApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodInfoRedisApplicationList contains a list of PodInfoRedisApplication
type PodInfoRedisApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodInfoRedisApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodInfoRedisApplication{}, &PodInfoRedisApplicationList{})
}

func (pira *PodInfoRedisApplication) RedisDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pira.Namespace,
			Name:      fmt.Sprintf("%v-%v", pira.Name, "redis"),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: pira.labels("redis")},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: pira.labels("redis")},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "redis",
							Image:   "public.ecr.aws/docker/library/redis:latest",
							Command: []string{"redis-server"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromString("redis"),
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"redis-cli", "ping"},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
							},
						},
					},
				},
			},
		},
	}
}

func (pira *PodInfoRedisApplication) RedisService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pira.Namespace,
			Name:      fmt.Sprintf("%v-%v", pira.Name, "redis"),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: pira.labels("redis"),
			Ports: []corev1.ServicePort{{
				Name:       "redis",
				Port:       6379,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromString("redis"),
			}},
		},
	}
}

func (pira *PodInfoRedisApplication) PodInfoDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pira.Namespace,
			Name:      fmt.Sprintf("%v-%v", pira.Name, "podinfo"),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pira.Spec.ReplicaCount,
			Selector: &metav1.LabelSelector{MatchLabels: pira.labels("podinfo")},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: pira.labels("podinfo")},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							// TODO configure: liveness/readiness probes, Deployment strategy, PDBs, minready, etc
							Name:  "podinfo",
							Image: fmt.Sprintf("%v:%v", pira.Spec.Image.Repository, pira.Spec.Image.Tag),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: pira.Spec.MemoryLimit,
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: pira.Spec.Resources.CpuRequest,
								},
							},
							Command: []string{"./podinfo", "--port=9898"}, // hardcoding port for now. generate ports dynamically to avoid overlap?
							Env: []corev1.EnvVar{
								{
									Name:  "PODINFO_UI_COLOR",
									Value: pira.Spec.UI.Color,
								},
								{
									Name:  "PODINFO_UI_MESSAGE",
									Value: pira.Spec.UI.Message,
								},
								{
									Name:  "PODINFO_CACHE_SERVER",
									Value: fmt.Sprintf("tcp://%v-redis:6379", pira.Name),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "podinfo",
									ContainerPort: 9898,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (pira *PodInfoRedisApplication) PodInfoService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pira.Namespace,
			Name:      fmt.Sprintf("%v-%v", pira.Name, "podinfo"),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: pira.labels("podinfo"),
			Ports: []corev1.ServicePort{
				{
					Name:     "podinfo",
					Port:     9898,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func (pira *PodInfoRedisApplication) labels(application string) map[string]string {
	return map[string]string{
		fmt.Sprintf("%v/%v", GroupVersion.Group, reflect.TypeOf(pira).Elem().Name()): string(pira.UID),
		fmt.Sprintf("%v/Application", GroupVersion.Group):                            application,
	}
}
