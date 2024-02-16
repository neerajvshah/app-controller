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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1 "neeraj.angi/app-operator/api/v1"
	v1 "neeraj.angi/app-operator/api/v1"
	"neeraj.angi/app-operator/util/kubeclient"
)

// PodInfoRedisApplicationReconciler reconciles a PodInfoRedisApplication object
type PodInfoRedisApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.neeraj.angi,resources=podinforedisapplication,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.neeraj.angi,resources=podinforedisapplication/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.neeraj.angi,resources=podinforedisapplication/finalizers,verbs=update
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;watch;list;create;update;delete
// +kubebuilder:rbac:groups="",resources=service,verbs=get;watch;list;create;update;delete
func (r *PodInfoRedisApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pira := &v1.PodInfoRedisApplication{}
	if err := r.Client.Get(ctx, req.NamespacedName, pira); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	objs := []client.Object{pira.PodInfoService(), pira.PodInfoDeployment()}
	if pira.Spec.Redis.Enabled {
		objs = append(objs, pira.RedisService(), pira.RedisDeployment())
	} else {
		for _, obj := range []client.Object{pira.RedisDeployment(), pira.RedisService()} {
			if err := r.Client.Delete(ctx, obj, &client.DeleteOptions{}); client.IgnoreNotFound(err) != nil {
				return reconcile.Result{}, fmt.Errorf("deleting: %v", err)
			}
		}
	}
	for _, obj := range objs {
		if err := controllerutil.SetControllerReference(pira, obj, r.Scheme); err != nil {
			return reconcile.Result{}, fmt.Errorf("setting owner reference: %v", err)
		}
		if err := kubeclient.Apply(ctx, r.Client, obj); err != nil {
			return reconcile.Result{}, fmt.Errorf("applying: %v", err)
		}
	}

	if err := r.Client.Get(ctx, req.NamespacedName, pira); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodInfoRedisApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.PodInfoRedisApplication{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
