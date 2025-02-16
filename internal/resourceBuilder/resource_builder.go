package resourceBuilder

import (
	appsv1 "github.com/mortada-codes/pot-app-operator/api/v1"
	appsv2 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // Import for ObjectMeta
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)


type ResourceBuilder interface {
	BuildDeployment(potApp *appsv1.PotApplication) *appsv2.Deployment
	BuildService(potApp *appsv1.PotApplication) *corev1.Service
}

func NewResourceBuilder(client client.Client,schema *runtime.Scheme) ResourceBuilder {
	return &potApplicationResourceBuilder{
		client: client,
		schema: schema,
	}
}


type potApplicationResourceBuilder struct {
	client client.Client
	schema *runtime.Scheme
}

func (builder *potApplicationResourceBuilder) BuildDeployment(potApp *appsv1.PotApplication) *appsv2.Deployment {
	
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
	return deployment

}

func (builder *potApplicationResourceBuilder) BuildService(potApp *appsv1.PotApplication) *corev1.Service {
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

	return service
}