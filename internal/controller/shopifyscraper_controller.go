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
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lukemcewencomv1 "github.com/lmcewen9/shopify-operator/api/v1"
	model "github.com/lmcewen9/shopify-operator/internal/scraper"
)

var config *rest.Config

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

	if _, inCluster := os.LookupEnv("KUBERNETES_SERVICE_HOST"); inCluster {
		config, _ = rest.InClusterConfig()
	}
	if config == nil {
		server := os.Getenv("MASTERURL")
		certificateAuthorityData := os.Getenv("CADATA")
		clientCertificateData := os.Getenv("CERTDATA")
		clientCertificateKeyData := os.Getenv("CLIENTDATA")

		// Decode base64 certificates
		caCertBytes, err := base64.StdEncoding.DecodeString(certificateAuthorityData)
		if err != nil {
			logger.Error(err, "Failed to decode CA certificate")
		}

		clientCertBytes, err := base64.StdEncoding.DecodeString(clientCertificateData)
		if err != nil {
			logger.Error(err, "Failed to decode client certificate")
		}

		clientKeyBytes, err := base64.StdEncoding.DecodeString(clientCertificateKeyData)
		if err != nil {
			logger.Error(err, "Failed to decode client key")
		}

		// Load CA certificate
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertBytes) {
			logger.Error(err, "Failed to append CA certificate")
		}

		// Create Kubernetes config
		config = &rest.Config{
			Host: server,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   caCertBytes,
				CertData: clientCertBytes,
				KeyData:  clientKeyBytes,
			},
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "error creating kubernetes clientset")
	}

	var scraper lukemcewencomv1.ShopifyScraper
	if err := r.Get(ctx, req.NamespacedName, &scraper); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	data, err := Scrape(&scraper)
	if err != nil {
		logger.Error(err, "did not return data")
	}
	encodedData := base64.StdEncoding.EncodeToString([]byte(strings.Join(data, "")))

	// FOR TESTING AUTOLOAD DATABASE
	/* tmpEncodedData := base64.StdEncoding.EncodeToString([]byte(strings.Join(data[:len(data)*97/100], "")))
	tmpCommand := fmt.Sprintf("echo %s > /shopify/%s", tmpEncodedData, req.Name)
	if _, err = execInPod(clientset, config, "shopify-operator-system", "shopify-operator-pod", "shopify-operator-pod", tmpCommand); err != nil {
		logger.Error(err, "could not autoload data")
	} else {
		logger.Info("successfully autoloaded data")
	} */
	// END OF AUTOLOAD DATABASE
	pullCommand := fmt.Sprintf("cat /shopify/%s", req.Name)
	encodedOldData, err := execInPod(clientset, config, "shopify-operator-system", "shopify-operator-pod", "shopify-operator-pod", pullCommand)
	if err != nil {
		if scraper.Status.NotFirst {
			logger.Error(err, "could not pull data")
		} else {
			scraper.Status.NotFirst = true
		}
	}

	decodedData, decodedOldData, err := compareBase64Decode(encodedData, string(encodedOldData))
	if err != nil {
		logger.Error(err, "error in decode")
		return ctrl.Result{}, err
	}
	if decodedData != nil || decodedOldData != nil {
		var difference []string
		for _, s := range decodedData {
			if _, exists := decodedOldData[s]; !exists {
				difference = append(difference, s)
			}
		}

		if err = sendWebhook(difference); err != nil {
			logger.Error(err, "failed to send webhook")
		}

		pushCommand := fmt.Sprintf("echo %s > /shopify/%s", encodedData, req.Name)
		if _, err = execInPod(clientset, config, "shopify-operator-system", "shopify-operator-pod", "shopify-operator-pod", pushCommand); err != nil {
			logger.Error(err, "could not push data")
		} else {
			logger.Info("successfully pushed data")
		}

		return ctrl.Result{RequeueAfter: time.Duration(*scraper.Spec.WatchTime) * time.Second}, nil
	} else {
		logger.Info("No difference in data")
		return ctrl.Result{RequeueAfter: time.Duration(*scraper.Spec.WatchTime) * time.Second}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShopifyScraperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lukemcewencomv1.ShopifyScraper{}).
		Named("shopifyscraper").
		Complete(r)
}

func compareBase64Decode(new, old string) ([]string, map[string]struct{}, error) {
	decoded1, err1 := base64.StdEncoding.DecodeString(new)
	decoded2, err2 := base64.StdEncoding.DecodeString(old)

	if err1 != nil || err2 != nil {
		return nil, nil, fmt.Errorf("error decoding base64: %v, %v", err1, err2)
	}

	if string(decoded1) == string(decoded2) {
		return nil, nil, nil
	}
	return formatStrArr(convertByteToStringArray(decoded1)), createSet(formatStrArr(convertByteToStringArray(decoded2))), nil
}

func createSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, str := range slice {
		set[str] = struct{}{} // Use an empty struct to save memory
	}
	return set
}

func formatStrArr(strArr []string) []string {
	return strings.Split(strings.Join(strArr, ""), ",")
}

func convertByteToStringArray(b []byte) []string {
	strArray := make([]string, len(b))
	for i, k := range b {
		strArray[i] = string(k)
	}
	return strArray
}

func Scrape(scraper *lukemcewencomv1.ShopifyScraper) ([]string, error) {
	var data []string
	var err error
	page := 1
	for {
		s, err := model.FetchShopify(&model.Configuration{
			URL: scraper.Spec.Url,
		}, page)

		if s == nil {
			break
		}
		if err != nil {
			return data, err
		}
		data = append(data, s...)
		page++
	}

	return data, err
}

func execInPod(clientset *kubernetes.Clientset, config *rest.Config, namespace, podName, containerName, command string) ([]byte, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"sh", "-c", command},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, req.URL())
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func sendWebhook(d []string) error {

	url := "http://localhost:8888/scraper-webhook"

	data := WebHookData{Message: d}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Context-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if e := resp.Body.Close(); err != nil {
			err = e
		}
	}()

	return nil
}
