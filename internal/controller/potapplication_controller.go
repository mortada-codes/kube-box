/*
Copyright 2025.

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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/mortada-codes/pot-app-operator/api/v1"
	"github.com/mortada-codes/pot-app-operator/internal/resourceBuilder"

	appsv2 "k8s.io/api/apps/v1" // Import for Deployment
	corev1 "k8s.io/api/core/v1" // Import for Pod/Container/Ports
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	// Import for ObjectMeta
)

// PotApplicationReconciler reconciles a PotApplication object
type PotApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	resourceBuilder resourceBuilder.ResourceBuilder
}

// +kubebuilder:rbac:groups=apps.example.com,resources=potapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=potapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=potapplications/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PotApplication object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *PotApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var potApp appsv1.PotApplication
	if err := r.Get(ctx, req.NamespacedName, &potApp); err != nil {
		if errors.IsNotFound(err) {
			log.Info("PotApplication resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PotApplication")
		return ctrl.Result{}, err
	}

	deployment := r.resourceBuilder.BuildDeployment(&potApp)
	service := r.resourceBuilder.BuildService(&potApp)


	if err := ctrl.SetControllerReference(&potApp, deployment, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference on Deployment")
		return ctrl.Result{}, err
	}
	if err := ctrl.SetControllerReference(&potApp, service, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference on Service")
		return ctrl.Result{}, err
	}



	foundDeployment := &appsv2.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
	}, foundDeployment)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new Deployment")
			err = r.Create(ctx, deployment)
			if err != nil {
				log.Error(err, "Failed to create new Deployment")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	if !equality.Semantic.DeepEqual(deployment.Spec, foundDeployment.Spec) {
		log.Info("Deployment spec is different. Updating Deployment")
		updatedDeployment := foundDeployment.DeepCopy()
		updatedDeployment.Spec = deployment.Spec
		err = r.Update(ctx, updatedDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment")
			return ctrl.Result{}, err
		}
		log.Info("Deployment spec updated successfully", "deploymentName", deployment.Name)
		return ctrl.Result{Requeue: true}, nil
	}
	
	foundService := &corev1.Service{}

	err = r.Get(ctx, types.NamespacedName{
		Name:      service.Name,
		Namespace: service.Namespace,
	},foundService)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new Service")
			err = r.Create(ctx, service)
			if err != nil {
				log.Error(err, "Failed to create new Service")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	if !equality.Semantic.DeepEqual(service.Spec, foundService.Spec) {
		log.Info("Service spec is different. Updating Service")
		updatedService := foundService.DeepCopy()
		updatedService.Spec = service.Spec
		err = r.Update(ctx, updatedService)
		if err != nil {
			log.Error(err, "Failed to update Service")
			return ctrl.Result{}, err
		}
		log.Info("Service spec updated successfully", "serviceName", service.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Deployment already exists", "deployment", foundDeployment)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PotApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.resourceBuilder = resourceBuilder.NewResourceBuilder(r.Client, r.Scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.PotApplication{}).
		Named("potapplication").
		Complete(r)
}

