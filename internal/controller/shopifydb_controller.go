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
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lukemcewencomv1 "github.com/lmcewen9/shopify-crd/api/v1"
)

var ServiceName string

// ShopifyDBReconciler reconciles a ShopifyDB object
type ShopifyDBReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=lukemcewen.com,resources=shopifydbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lukemcewen.com,resources=shopifydbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=lukemcewen.com,resources=shopifydbs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ShopifyDB object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *ShopifyDBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	db := &lukemcewencomv1.ShopifyDB{}
	if err := r.Get(ctx, req.NamespacedName, db); err != nil {
		logger.Error(err, "Could not fetch PostgresDB CRD")
		return ctrl.Result{}, nil
	}

	db.SetDefaults()
	ServiceName = req.Name

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name + "-pv",
			Namespace: req.Namespace,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(db.Spec.StorageSize),
			},
			VolumeMode:                    ptr(corev1.PersistentVolumeBlock),
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              "manual",
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/data/" + req.Name,
				},
			},
		},
	}
	if err := r.Create(ctx, pv); err != nil {
		logger.Error(err, "PV already exists or error")
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name + "-pv-claim",
			Namespace: req.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &[]string{"manual"}[0],
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(db.Spec.StorageSize),
				},
			},
		},
	}
	if err := r.Create(ctx, pvc); err != nil {
		logger.Error(err, "PVC already exists or error")
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name + "-deployment",
			Namespace: req.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &db.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": req.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": req.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  req.Name,
							Image: db.Spec.Image,
							Ports: []corev1.ContainerPort{{ContainerPort: 5432}},
							Env: []corev1.EnvVar{
								{Name: "DB_USER", Value: dbUser},
								{Name: "DB_PASSWORD", Value: dbPass},
								{Name: "DB_NAME", Value: dbName},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shopifydb-storage",
									MountPath: "/var/lib/postgresql/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "shopifydb-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: req.Name + "pv-claim",
								},
							},
						},
					},
				},
			},
		},
	}
	if err := r.Create(ctx, deployment); err != nil {
		logger.Error(err, "Deployment already exists or error")
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name + "-svc",
			Namespace: req.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": req.Name},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
				},
			},
		},
	}
	if err := r.Create(ctx, svc); err != nil {
		logger.Error(err, "Service already exists or error")
	}

	return ctrl.Result{}, nil
}

func ptr[T any](v T) *T {
	return &v
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShopifyDBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lukemcewencomv1.ShopifyDB{}).
		Named("shopifydb").
		Complete(r)
}

func getServiceName() string {
	return ServiceName
}
