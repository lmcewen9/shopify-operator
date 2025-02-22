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
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lukemcewencomv1 "github.com/lmcewen9/shopify-crd/api/v1"
)

// DiscordBotReconciler reconciles a DiscordBot object
type DiscordBotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=lukemcewen.com,resources=discordbots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lukemcewen.com,resources=discordbots/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=lukemcewen.com,resources=discordbots/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DiscordBot object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *DiscordBotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var bot lukemcewencomv1.DiscordBot
	if err := r.Get(ctx, req.NamespacedName, &bot); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if bot.Spec.Token == "" {
		logger.Error(errors.New("deed discord token"), "Bot needs Discord Token")
	}

	if !bot.Status.Running {
		go startDiscordBot(bot.Spec.Token, ctx)
		bot.Status.Running = true
		if err := r.Status().Update(ctx, &bot); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DiscordBotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lukemcewencomv1.DiscordBot{}).
		Named("discordbot").
		Complete(r)
}

func startDiscordBot(token string, ctx context.Context) {
	logger := log.FromContext(ctx)
	dg, err := discordgo.New("ControllerBot" + token)
	if err != nil {
		logger.Error(err, "Error creating Discord session")
		return
	}

	dg.AddHandler(messageHandler)
	if err := dg.Open(); err != nil {
		logger.Error(err, "Error opening Discord connection")
		return
	}

	logger.Info("Bot is running. Pless CTRL+C to stop.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	dg.Close()
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, "!ping") {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
}
