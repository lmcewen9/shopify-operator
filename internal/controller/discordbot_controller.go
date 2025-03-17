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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/exp/rand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type WebHookData struct {
	Message []string `json:"message"`
}

var (
	channelID     string
	Prefix        = "!"
	dg            *discordgo.Session
	botReconciler *DiscordBotReconciler
	guildID       string
)

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
		logger.Error(errors.New("need discord token"), "Bot needs Discord Token")
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
	botReconciler = r
	return ctrl.NewControllerManagedBy(mgr).
		For(&lukemcewencomv1.DiscordBot{}).
		Named("discordbot").
		Complete(r)
}

func ptr[T any](value T) *T {
	return &value
}

func (r *DiscordBotReconciler) createScraper(name string) error {
	scraper := &lukemcewencomv1.ShopifyScraper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "shopify-crd-system",
		},
		Spec: lukemcewencomv1.ShopifyScraperSpec{
			Name:      name,
			Url:       name,
			WatchTime: ptr(int32(86400)), // seconds in a day
		},
	}

	if err := r.Create(context.TODO(), scraper); err != nil {
		return err
	}

	return nil
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.FromContext(context.TODO())

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			logger.Error(err, "failed to close request body")
		}
	}()

	var data WebHookData
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if data.Message != nil {
		logger.Info("received webhook data")
	}

	w.WriteHeader(http.StatusOK)

	for dg == nil {
		time.Sleep(2 * time.Second)
	}

	if err := eventHandler(dg, data); err != nil {
		logger.Error(err, "failed to start eventHandler")
	}
}

func SetupDataTransferWebhookWithManager(mgr ctrl.Manager) {
	logger := mgr.GetLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/scraper-webhook", webhookHandler)

	go func() {
		logger.Info("webhook server listening on :8888")
		if err := http.ListenAndServe(":8888", mux); err != nil {
			logger.Error(err, "unable to create webhook", "webhook", "DataTransfer")
			os.Exit(1)
		}
	}()
}

func startDiscordBot(token string, ctx context.Context) {
	logger := log.FromContext(ctx)
	var err error

	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		logger.Error(err, "Error creating Discord session")
		return
	}

	dg.AddHandler(messageHandler)

	if err = dg.Open(); err != nil {
		logger.Error(err, "Error opening Discord connection")
		return
	}

	logger.Info("Bot is running. Pless CTRL+C to stop.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	if err = dg.Close(); err != nil {
		logger.Error(err, "failed to close *discordgo.Session")
	}
}

func sendMessage(s *discordgo.Session, message []string, roleID string) error {
	if _, err := s.ChannelMessageSend(channelID, fmt.Sprintf("<@&%s> %s", roleID, strings.Join(message, ""))); err != nil {
		return err
	}
	return nil
}

func eventHandler(s *discordgo.Session, data WebHookData) error {
	message := data.Message
	re := regexp.MustCompile(`https?:\/\/([^\/]+)`)
	roleID := roleExists(s, guildID, re.FindStringSubmatch(message[0])[1])
	if len(strings.Join(message, "")) > 2000-(len(roleID)+2) {
		var smallMessage []string
		count := 0
		for {
			smallMessage = append(smallMessage, message[count])
			if len(strings.Join(smallMessage, "")) > 2000-(len(roleID)+1) {
				smallMessage = smallMessage[:len(smallMessage)-1]
				if err := sendMessage(s, smallMessage, roleID); err != nil {
					return err
				}
				smallMessage = nil
				count--

			}

			if count == len(message)-1 {
				if err := sendMessage(s, smallMessage, roleID); err != nil {
					return err
				}
				return nil
			}
			count++
		}
	}

	if err := sendMessage(s, message, roleID); err != nil {
		return err
	}
	return nil
}

func randomHexColor() *int {
	rand.New(rand.NewSource(uint64(time.Now().UnixNano()))) // Seed for randomness
	hexValue := rand.Intn(0xFFFFFF)                         // Generate random hex value
	return &hexValue
}

func roleExists(s *discordgo.Session, guildID, roleName string) string {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return ""
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			return role.ID
		}
	}
	return ""
}

func createRole(s *discordgo.Session, guildID, roleName, userID string) error {
	roleID := roleExists(s, guildID, roleName)
	if roleID == "" {
		newRole, err := s.GuildRoleCreate(guildID, &discordgo.RoleParams{
			Name:  roleName,
			Color: randomHexColor(),
			Hoist: ptr(true),
		})
		if err != nil {
			return err
		}
		roleID = newRole.ID
	}

	if err := s.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
		return err
	}

	return nil
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger := log.FromContext(context.TODO())

	if m.Author.Bot {
		return
	}
	channelID = m.ChannelID
	// Check if the message starts with the prefix
	if !strings.HasPrefix(m.Content, Prefix) {
		return
	}

	// Extract command (removing prefix and splitting arguments)
	args := strings.Fields(m.Content[len(Prefix):])
	if len(args) == 0 {
		return
	}
	command := args[0]

	// Handle commands using switch
	var err error
	switch command {
	case "ping":
		if _, err = s.ChannelMessageSend(m.ChannelID, "Pong! 🏓"); err != nil {
			logger.Error(err, "failed to send !ping response")
		}

	case "hello":
		if _, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello, %s! 👋", m.Author.Username)); err != nil {
			logger.Error(err, "failed to send !hello response")
		}

	case "add":
		createScraperErr := botReconciler.createScraper(args[1])
		if createScraperErr != nil {
			if _, err = s.ChannelMessageSend(m.ChannelID, "unable to create scraper or scraper already exists"); err != nil {
				logger.Error(err, "failed to send message for createScraper()")
			}
		}
		guildID = m.GuildID
		if createScraperErr == nil {
			if err = createRole(s, guildID, args[1], m.Author.ID); err != nil {
				if _, err = s.ChannelMessageSend(m.ChannelID, "failed to assign you to role"); err != nil {
					logger.Error(err, "failed to send message for createRole()")
				}
			}
		}

	case "help":
		helpMessage := "**Available Commands:**\n" +
			"`!ping` - Responds with Pong! 🏓\n" +
			"`!hello` - Greets you 👋\n" +
			"`!add <store domain>` - i.e. `!add store.example.com`, creates Shopify Scraper and adds user to role"
		if _, err = s.ChannelMessageSend(m.ChannelID, helpMessage); err != nil {
			logger.Error(err, "failed to send !help response")
		}

	default:
		if _, err = s.ChannelMessageSend(m.ChannelID, "Unknown command! Type `!help` for a list of commands."); err != nil {
			logger.Error(err, "failed to send ! response")
		}
	}
}
