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
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lukemcewencomv1 "github.com/lmcewen9/shopify-crd/api/v1"
	model "github.com/lmcewen9/shopify-crd/scraper"
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

	var scraper lukemcewencomv1.ShopifyScraper
	if err := r.Get(ctx, req.NamespacedName, &scraper); err != nil {
		return ctrl.Result{RequeueAfter: time.Duration(*scraper.Spec.WatchTime) * time.Second}, client.IgnoreNotFound(err)
	}

	checkDatabase(ctx, &scraper)

	return ctrl.Result{RequeueAfter: time.Duration(*scraper.Spec.WatchTime) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShopifyScraperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lukemcewencomv1.ShopifyScraper{}).
		Named("shopifyscraper").
		Complete(r)
}

func Scrape(ctx context.Context, scraper *lukemcewencomv1.ShopifyScraper) []string {
	logger := log.FromContext(ctx)

	var data []string
	page := 1
	for {
		s, err := model.FetchShopify(&model.Configuration{
			URL: scraper.Spec.Url,
		}, page)

		if s == nil {
			break
		}
		if err != nil {
			logger.Error(err, err.Error())
		}
		data = append(data, s...)
		page++
	}

	return data
}

func CreateTable(ctx context.Context, db *pgx.Conn, scraper *lukemcewencomv1.ShopifyScraper) error {
	if _, err := db.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS {$1} (id INT PRIMARY KEY, data TEXT)`, scraper.Spec.Name); err != nil {
		return err
	}
	return nil
}

func checkDatabase(ctx context.Context, scraper *lukemcewencomv1.ShopifyScraper) ([]string, error) {
	logger := log.FromContext(ctx)

	connection := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("DB_PASSWORD"),
		getServiceName(),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := pgx.Connect(ctx, connection)
	if err != nil {
		logger.Error(err, "failed to connect to database")
	}
	defer db.Close(ctx)

	if err = CreateTable(ctx, db, scraper); err != nil {
		logger.Error(err, "failed to create table")
	}

	var existingData []string
	if err = db.QueryRow(context.Background(), "SELECT data FROM {$1} ORDER BY id DESC LIMIT 1", scraper.Spec.Name).Scan(&existingData); err != nil && err != sql.ErrNoRows {
		logger.Error(err, "falied to query database")
	}

	newData := Scrape(ctx, scraper)
	if !reflect.DeepEqual(newData, existingData) {

		existingDataMap := make(map[string]bool)
		var differences []string

		for _, item := range existingData {
			existingDataMap[item] = true
		}

		for _, item := range newData {
			if !existingDataMap[item] {
				differences = append(differences, item)
			}
		}

		if _, err = db.Exec(ctx, "DROP TABLE IF EXISTS {$1}", scraper.Spec.Name); err != nil {
			logger.Error(err, "failed to drop table")
		}
		if err = CreateTable(ctx, db, scraper); err != nil {
			logger.Error(err, "failed to create table")
		}
		if _, err = db.Exec(ctx, "INSERT INTO {$1} (data) VALUES ({$2})", scraper.Spec.Name, newData); err != nil {

		}

		return differences, nil
	}

	return nil, err
}
