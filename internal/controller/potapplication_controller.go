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
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/mortada-codes/pot-app-operator/api/v1"

	appsv2 "k8s.io/api/apps/v1" // Import for Deployment
	corev1 "k8s.io/api/core/v1" // Import for Pod/Container/Ports
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // Import for ObjectMeta
)

// PotApplicationReconciler reconciles a PotApplication object
type PotApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

	deployment := r.desiredDeployment(&potApp)

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

	service, err := r.constructService(&potApp)
	if err != nil {
		log.Error(err, "Failed to construct service")
		return ctrl.Result{}, err
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

	log.Info("Deployment already exists", "deployment", foundDeployment)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PotApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.PotApplication{}).
		Named("potapplication").
		Complete(r)
}

func (r *PotApplicationReconciler) desiredDeployment(potApp *appsv1.PotApplication) *appsv2.Deployment {

	int32Ptr := func(i int32) *int32 {
		val := i    
		return &val 
	}
	deployment := &appsv2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      potApp.Name,
			Namespace: potApp.Namespace,
		},
		Spec: appsv2.DeploymentSpec{
			Replicas: int32Ptr(potApp.Spec.Replicas), 
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": potApp.Name, 
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": potApp.Name, 
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web", 
							Image: potApp.Spec.Image, 
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80, 
									Name: "http",
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(potApp, deployment, r.Scheme)
	return deployment
}


func (r *PotApplicationReconciler) constructService(potApp *appsv1.PotApplication) (*corev1.Service, error) {
	
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      potApp.Name + "-service", 
			Namespace: potApp.Namespace,
			Labels:    potApp.Labels, 
		},
		Spec: corev1.ServiceSpec{
			Selector: potApp.Labels, 
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,        
					TargetPort: intstr.FromInt(8080), 
					NodePort:   30080,     
				},
			},
			Type: corev1.ServiceTypeNodePort, 
		},
	}

	// Set PotApplication instance as the owner and controller of the Service
	ctrl.SetControllerReference(potApp, service, r.Scheme)
	return service, nil
}

