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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodInfoRedisApplicationSpec defines the desired state of PodInfoRedisApplication
type PodInfoRedisApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PodInfoRedisApplication. Edit podinforedisapplication_types.go to remove/update
	Foo string `json:"foo,omitempty"`

	// TODO evaluate use of omitempty
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
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type UI struct {
	Color   string `json:"color,omitempty"`
	Message string `json:"message,omitempty"`
}

type Redis struct {
	Enabled bool `json:"enabled,omitempty"`
}

// PodInfoRedisApplicationStatus defines the observed state of PodInfoRedisApplication
type PodInfoRedisApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PodInfoRedisApplication is the Schema for the podinforedisapplications API
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

func (app *PodInfoRedisApplication) AppDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: app.Namespace,
			Name:      app.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.ReplicaCount,
			Selector: &metav1.LabelSelector{MatchLabels: app.labels()},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: app.labels()},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							// TODO configure: liveness/readiness probes, Deployment strategy, PDBs, minready, etc
							Name:  "podinfo",
							Image: fmt.Sprintf("%v:%v", app.Spec.Image.Repository, app.Spec.Image.Tag),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: app.Spec.MemoryLimit,
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: app.Spec.Resources.CpuRequest,
								},
							},
							Command: []string{"./podinfo", "--port=9898"}, // hardcoding port for now. generate ports dynamically to avoid overlap?
							Env: []corev1.EnvVar{
								{
									Name:  "PODINFO_UI_COLOR",
									Value: app.Spec.UI.Color,
								},
								{
									Name:  "PODINFO_UI_MESSAGE",
									Value: app.Spec.UI.Message,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
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

func (app *PodInfoRedisApplication) Service() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: app.Namespace, Name: app.Name},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: app.labels(),
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     9898,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func (app *PodInfoRedisApplication) labels() map[string]string {
	return map[string]string{
		fmt.Sprintf("%v/%v", GroupVersion.Group, reflect.TypeOf(app).Elem().Name()): string(app.UID),
	}
}
