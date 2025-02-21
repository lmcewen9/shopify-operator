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
	"fmt"

	discord "github.com/bwmarrin/discordgo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	lukemcewencomv1 "github.com/lmcewen9/shopify-crd/api/v1"
)

// ShopifyScraperReconciler reconciles a ShopifyScraper object
type ShopifyScraperReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=lukemcewen.com,resources=shopifyscrapers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lukemcewen.com,resources=shopifyscrapers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=lukemcewen.com,resources=shopifyscrapers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ShopifyScraper object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *ShopifyScraperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("controller triggered")

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func report(reporter lukemcewencomv1.ShopifyScraper, pod corev1.Pod) {
	// report to slack
	log.Log.V(1).Info("Reporting to reporter", "name", reporter.Spec.Name, "endpoint", reporter.Spec.Report.Key)
	discordChannel := reporter.Spec.Report.Channel
	app := discord.New(reporter.Spec.Report.Key, discord.OptionDebug(true))

	message := fmt.Sprintf("New pod created: %s", pod.Name)
	msgText := discord.NewTextBlockObject("mrkdwn", message, false, false)
	msgSection := discord.NewSectionBlock(msgText, nil, nil)
	msg := discord.MsgOptionBlocks(
		msgSection,
	)
	fmt.Print(msg)
	log.Log.V(1).Info("Reporting", "message", "", "channel", discordChannel)
	_, _, _, err := app.SendMessage(discordChannel, msg)

	if err != nil {
		log.Log.V(1).Info(err.Error())
	}
}

func (r *ShopifyScraperReconciler) HandlePodEvents(pod client.Object) []reconcile.Request {
	if pod.GetNamespace() != "default" {
		return []reconcile.Request{}
	}
	namespaceName := types.NamespacedName{
		Namespace: pod.GetNamespace(),
		Name:      pod.GetName(),
	}

	var podObject corev1.Pod
	err := r.Get(context.Background(), namespaceName, &podObject)

	if err != nil {
		return []reconcile.Request{}
	}

	if len(podObject.Annotations) == 0 {
		log.Log.V(1).Info("No annotations set, so this pod is becoming a tracked one now", "pod", podObject.Name)
	} else if podObject.GetAnnotations()["exampleAnnotation"] == "lukemcewen.com" {
		log.Log.V(1).Info("Found a managed pod, lets report it", "pod", podObject.Name)
	} else {
		return []reconcile.Request{}
	}

	podObject.SetAnnotations(map[string]string{
		"exampleAnnotation": "lukemcewen.com",
	})

	if err := r.Update(context.TODO(), &podObject); err != nil {
		log.Log.V(1).Info("error trying to update pod", "err", err)
	}

	return []reconcile.Request{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShopifyScraperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lukemcewencomv1.ShopifyScraper{}).
		Watches(
			&source.Kind{Type: &corev1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(handler.MapFunc(r.HandlePodEvents)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named("shopifyscraper").
		Complete(r)
}
