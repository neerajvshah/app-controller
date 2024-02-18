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

package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "neeraj.angi/app-operator/api/v1"
)

var _ = Describe("PodInfoRedisApplication Controller", func() {
	var reconciler *PodInfoRedisApplicationReconciler

	var pira *v1.PodInfoRedisApplication
	var podInfoNn types.NamespacedName
	var redisNn types.NamespacedName
	var podInfoDeployment appsv1.Deployment
	var podInfoService corev1.Service
	var redisDeployment appsv1.Deployment
	var redisService corev1.Service
	ctx := context.Background()

	Context("When reconciling a resource", func() {
		BeforeEach(func() {
			pira = &v1.PodInfoRedisApplication{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-app",
				},
				Spec: v1.PodInfoRedisApplicationSpec{
					ReplicaCount: lo.ToPtr(int32(2)),
					Resources: v1.Resources{
						MemoryLimit: resource.MustParse("500M"),
						CpuRequest:  resource.MustParse("250M"),
					},
					Image: v1.Image{
						Repository: "test-repo",
						Tag:        "test-tag",
					},
					UI: v1.UI{
						Color:   "#321903",
						Message: "hello world",
					},
					Redis: v1.Redis{
						Enabled: false,
					},
				},
			}
			podInfoNn = types.NamespacedName{Namespace: pira.Namespace, Name: fmt.Sprintf("%s-%s", pira.Name, "podinfo")}
			redisNn = types.NamespacedName{Namespace: pira.Namespace, Name: fmt.Sprintf("%s-%s", pira.Name, "redis")}
			reconciler = &PodInfoRedisApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})
		It("should only create podInfo resources if redis is disabled and create redis resources when enabled", func() {
			Expect(k8sClient.Create(ctx, pira)).To(BeNil())
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pira.ObjectMeta.Namespace, Name: pira.ObjectMeta.Name},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoDeployment)).To(BeNil())
			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoService)).To(BeNil())
			Expect(errors.IsNotFound(k8sClient.Get(ctx, redisNn, &appsv1.Deployment{}))).To(BeTrue())
			Expect(errors.IsNotFound(k8sClient.Get(ctx, redisNn, &corev1.Service{}))).To(BeTrue())

			pira.Spec.Redis.Enabled = true
			Expect(k8sClient.Update(ctx, pira)).To(BeNil())
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pira.ObjectMeta.Namespace, Name: pira.ObjectMeta.Name},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoDeployment)).To(BeNil())
			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoService)).To(BeNil())
			Expect(k8sClient.Get(ctx, redisNn, &redisDeployment)).To(BeNil())
			Expect(k8sClient.Get(ctx, redisNn, &redisService)).To(BeNil())
		})
		It("should create redis resources if redis is enabled and then delete when disabled", func() {
			pira.Spec.Redis.Enabled = true
			Expect(k8sClient.Create(ctx, pira)).To(BeNil())
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pira.ObjectMeta.Namespace, Name: pira.ObjectMeta.Name},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoDeployment)).To(BeNil())
			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoService)).To(BeNil())
			Expect(k8sClient.Get(ctx, redisNn, &redisDeployment)).To(BeNil())
			Expect(k8sClient.Get(ctx, redisNn, &redisService)).To(BeNil())

			pira.Spec.Redis.Enabled = false
			Expect(k8sClient.Update(ctx, pira)).To(BeNil())
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pira.ObjectMeta.Namespace, Name: pira.ObjectMeta.Name},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoDeployment)).To(BeNil())
			Expect(k8sClient.Get(ctx, podInfoNn, &podInfoService)).To(BeNil())
			Expect(errors.IsNotFound(k8sClient.Get(ctx, redisNn, &appsv1.Deployment{}))).To(BeTrue())
			Expect(errors.IsNotFound(k8sClient.Get(ctx, redisNn, &corev1.Service{}))).To(BeTrue())
		})

		AfterEach(func() {
			// Validate PodInfo Deployment
			Expect(podInfoDeployment.OwnerReferences[0].UID).To(Equal(pira.UID))
			Expect(*podInfoDeployment.Spec.Replicas).To(Equal(*pira.Spec.ReplicaCount))
			// Ordered env var checking - if I had time I would use a map to avoid ordering constraints, which aren't necessary
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env[0].Name).To(Equal("PODINFO_UI_COLOR"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env[0].Value).To(Equal(pira.Spec.UI.Color))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env[1].Name).To(Equal("PODINFO_UI_MESSAGE"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env[1].Value).To(Equal(pira.Spec.UI.Message))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env[2].Name).To(Equal("PODINFO_CACHE_SERVER"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env[2].Value).To(Equal(fmt.Sprintf("tcp://%v-redis:6379", pira.Name)))

			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(9898)))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))

			Expect(*podInfoDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory()).To(Equal(pira.Spec.Resources.MemoryLimit))
			Expect(*podInfoDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu()).To(Equal(pira.Spec.Resources.CpuRequest))

			// Validate PodInfo Service
			Expect(podInfoService.OwnerReferences[0].UID).To(Equal(pira.UID))
			Expect(podInfoService.Spec.Selector).To(Equal(podInfoDeployment.Spec.Selector.MatchLabels))
			Expect(podInfoService.Spec.Ports[0].Port).To(Equal(int32(9898)))

			// Validate Redis Deployment
			Expect(redisDeployment.OwnerReferences[0].UID).To(Equal(pira.UID))
			Expect(redisDeployment.Spec.Template.Spec.Containers[0].Ports[0].Name).To(Equal("redis"))
			Expect(redisDeployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(6379)))
			Expect(redisDeployment.Spec.Template.Spec.Containers[0].Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))

			// Validate Redis Service
			Expect(redisService.OwnerReferences[0].UID).To(Equal(pira.UID))
			Expect(redisService.Spec.Selector).To(Equal(redisDeployment.Spec.Selector.MatchLabels))
			Expect(redisService.Spec.Ports[0].Port).To(Equal(int32(6379)))
			Expect(redisService.Spec.Ports[0].TargetPort.StrVal).To(Equal("redis"))

			By("Cleanup the specific resource instance PodInfoRedisApplication")
			Expect(k8sClient.Delete(ctx, pira)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &podInfoDeployment)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &podInfoService)).To(Succeed())
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &redisDeployment))).To(Succeed())
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, &redisService))).To(Succeed())
		})
	})
})
