/*
Copyright 2022 The Crossplane Authors.

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

package ctrldrift

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane/provider-driftprovider/apis/mlops/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-driftprovider/apis/v1alpha1"
	"github.com/crossplane/provider-driftprovider/internal/features"

	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	errNotCtrlDrift = "managed resource is not a CtrlDrift custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type NoOpService struct{}

var (
	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
)

// Setup adds a controller that reconciles CtrlDrift managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.CtrlDriftGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.CtrlDriftGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			logger:       o.Logger,
			newServiceFn: newNoOpService}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.CtrlDrift{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	logger       logging.Logger
	newServiceFn func(creds []byte) (interface{}, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.CtrlDrift)
	if !ok {
		return nil, errors.New(errNotCtrlDrift)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := c.newServiceFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, logger: c.logger}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service interface{}
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.CtrlDrift)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCtrlDrift)
	}
	c.logger.Debug(fmt.Sprintf("Observing: %+v", cr))

	//connect to kubernetes
	clientset, err := c.connect_kube_client()

	folder_path := "/var/data/"
	drift_data := "drift_data.csv"
	drifting := false

	resource_exists := false
	resource_uptodate := true

	if err != nil {
		c.logger.Debug("Error in connecting to kubernetes")
		c.logger.Debug(err.Error())
	}

	//check if drifting deployment is running
	deployments, err := clientset.AppsV1().Deployments("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Debug("Error in listing deployments")
		c.logger.Debug(err.Error())
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == "drift-deploy" {
			c.logger.Debug("Drift detection deployment already running")
			resource_exists = true
		}
	}

	//get all files in folder
	files, err := os.ReadDir(folder_path)

	if err != nil {
		c.logger.Debug("Error in reading directory")
		c.logger.Debug(err.Error())
	}

	for _, file := range files {
		if file.Name() == drift_data {
			c.logger.Debug("drift data file found")
			drifting = true
		}
	}
	if drifting {
		//check data drift length as parameter for retraining
		content, err := os.ReadFile(folder_path + drift_data)
		if err != nil {
			c.logger.Debug("Error in reading drifted data")
			c.logger.Debug(err.Error())
		}
		//count /n
		lines := strings.Split(string(content), "\n")
		c.logger.Debug(fmt.Sprintf("Number of lines in %s: %d", drift_data, len(lines)))

		if len(lines) > 3000 {
			//check if the new nodel has been trained on the new data

			c.logger.Debug("Data drift detected, retraining needed")

			//check if training job is running
			jobs, err := clientset.BatchV1().Jobs("default").List(ctx, metav1.ListOptions{})
			if err != nil {
				c.logger.Debug("Error in listing jobs")
				c.logger.Debug(err.Error())
			}
			//if no jobs are running, start training job
			if len(jobs.Items) == 0 {
				c.logger.Debug("Start training job")
				//create job
				training_job := get_training_job()

				_, err = clientset.BatchV1().Jobs("default").Create(ctx, training_job, metav1.CreateOptions{})
				if err != nil {
					c.logger.Debug("Error in creating training job")
					c.logger.Debug(err.Error())
				} else {
					c.logger.Debug("training job created")
				}
			}

			for _, job := range jobs.Items {
				if job.Name == "training-job" {
					c.logger.Debug("Training job already running")
				} else {
					c.logger.Debug("Start training job")
					//create job
					training_job := get_training_job()

					_, err = clientset.BatchV1().Jobs("default").Create(ctx, training_job, metav1.CreateOptions{})
					if err != nil {
						c.logger.Debug("Error in creating job")
						c.logger.Debug(err.Error())
					} else {
						c.logger.Debug("Job created")
					}

					//delete resource

				}
			}

		}
	}

	//check if conversion job is running
	jobs, err := clientset.BatchV1().Jobs("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Debug("Error in listing jobs")
		c.logger.Debug(err.Error())
	}

	for _, job := range jobs.Items {
		if job.Name == "converting-job" {
			//check if job is completed
			if job.Status.Succeeded == 1 {
				c.logger.Debug("Conversion job completed")
				//delete job
				delete_options := metav1.DeleteOptions{PropagationPolicy: &[]metav1.DeletionPropagation{"Background"}[0]}
				err = clientset.BatchV1().Jobs("default").Delete(ctx, job.Name, delete_options)
				if err != nil {
					c.logger.Debug("Error in deleting job")
					c.logger.Debug(err.Error())
				}
				//change model in deployment
				//TODO: change model in deployment
			} else {
				c.logger.Debug("Conversion job still running")
			}

		}
		//check if job is completed
		if job.Name == "training-job" {
			if job.Status.Succeeded == 1 {
				c.logger.Debug("Training job completed")
				//delete job and pod
				delete_options := metav1.DeleteOptions{PropagationPolicy: &[]metav1.DeletionPropagation{"Background"}[0]}
				err = clientset.BatchV1().Jobs("default").Delete(ctx, job.Name, delete_options)
				if err != nil {
					c.logger.Debug("Error in deleting job")
					c.logger.Debug(err.Error())
				}
				//reload drift deployment
				resource_uptodate = false

				//convert model to tflite running convert
				convert_job := get_converting_job()

				_, err = clientset.BatchV1().Jobs("default").Create(ctx, convert_job, metav1.CreateOptions{})
				if err != nil {
					c.logger.Debug("Error in creating job")
					c.logger.Debug(err.Error())
				} else {
					c.logger.Debug("conversion job created")
				}
			} else {
				c.logger.Debug("Training job still running")
			}
		}
	}

	c.logger.Debug(fmt.Sprintf("Drifting: %t", drifting))

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: resource_exists,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: resource_uptodate,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.CtrlDrift)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCtrlDrift)
	}

	c.logger.Debug(fmt.Sprintf("Creating: %+v", cr))

	//connect to kubernetes
	clientset, err := c.connect_kube_client()

	if err != nil {
		c.logger.Debug("Error in connecting to kubernetes")
		c.logger.Debug(err.Error())
	}

	//create drift deployment

	deployment := get_drift_detection_deployment()

	_, err = clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		c.logger.Debug("Error in creating drift deployment")
		c.logger.Debug(err.Error())
	}

	//create inference deployment

	deployment = get_tflite_deployment()

	_, err = clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		c.logger.Debug("Error in creating tflite deployment")
		c.logger.Debug(err.Error())
	}

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.CtrlDrift)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCtrlDrift)
	}
	c.logger.Debug(fmt.Sprintf("Updating: %+v", cr))

	//connect to kubernetes
	clientset, err := c.connect_kube_client()

	if err != nil {
		c.logger.Debug("Error in connecting to kubernetes")
		c.logger.Debug(err.Error())
	}

	//restart deployment drift detection

	err = clientset.AppsV1().Deployments("default").Delete(ctx, "drift-deploy", metav1.DeleteOptions{})
	if err != nil {
		c.logger.Debug("Error in deleting drift deployment")
		c.logger.Debug(err.Error())
	}

	deployment := get_drift_detection_deployment()

	_, err = clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		c.logger.Debug("Error in creating drift deployment")
		c.logger.Debug(err.Error())
	}

	c.logger.Debug("Deployment drift restarted")

	//restart deployment inference

	err = clientset.AppsV1().Deployments("default").Delete(ctx, "python-tflite-deploy", metav1.DeleteOptions{})

	if err != nil {
		c.logger.Debug("Error in deleting tflite deployment")
		c.logger.Debug(err.Error())
	}

	deployment = get_tflite_deployment()

	_, err = clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})

	if err != nil {
		c.logger.Debug("Error in creating tflite deployment")
		c.logger.Debug(err.Error())
	}

	c.logger.Debug("Deployment tflite restarted")

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.CtrlDrift)
	if !ok {
		return errors.New(errNotCtrlDrift)
	}

	c.logger.Debug(fmt.Sprintf("Deleting: %+v", cr))

	//connect to kubernetes
	clientset, err := c.connect_kube_client()

	if err != nil {
		c.logger.Debug("Error in connecting to kubernetes")
		c.logger.Debug(err.Error())
	}

	//delete deployment drift detection

	err = clientset.AppsV1().Deployments("default").Delete(ctx, "drift-deploy", metav1.DeleteOptions{})
	if err != nil {
		c.logger.Debug("Error in deleting drift deployment")
		c.logger.Debug(err.Error())
	}

	//delete deployment inference

	err = clientset.AppsV1().Deployments("default").Delete(ctx, "python-tflite-deploy", metav1.DeleteOptions{})
	if err != nil {
		c.logger.Debug("Error in deleting tflite deployment")
		c.logger.Debug(err.Error())
	}

	return nil
}
