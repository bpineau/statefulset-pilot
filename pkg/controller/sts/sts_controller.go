/*
Copyright 2018 The Kubernetes Authors.

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

package sts

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/bpineau/statefulset-pilot/pkg/hooks"
	hookfactory "github.com/bpineau/statefulset-pilot/pkg/hooks/factory"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// StatefulsetPilotLabelKey is the sts label that name the hook we'll
	// call between rollout steps. We ignore statefulsets without this label.
	StatefulsetPilotLabelKey = "dd-statefulset-pilot"

	retryInterval = 30 * time.Second

	pred = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, ok := e.MetaNew.GetLabels()[StatefulsetPilotLabelKey]
			if !ok {
				return false
			}
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Meta.GetLabels()[StatefulsetPilotLabelKey]
			return ok
		},
	}
)

// Add creates a new sts Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSts{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		log:    logf.Log.WithName("reconcile"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("sts-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to statefulsets
	err = c.Watch(
		&source.Kind{Type: &appsv1.StatefulSet{}},
		&handler.EnqueueRequestForObject{},
		pred,
	)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSts{}

// ReconcileSts reconciles a statefulset object
type ReconcileSts struct {
	client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

// Reconcile make cluster changes according to the statefulset spec.
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=statefulset,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileSts) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Fetch the sts instance
	instance := &appsv1.StatefulSet{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Fetch the hook named in StatefulsetPilotLabelKey label
	hook, err := hookfactory.Get(instance, StatefulsetPilotLabelKey)
	if err != nil {
		r.log.Error(err, "unsupported or no hook type, ignoring this sts")
		return reconcile.Result{}, nil
	}

	// We mandate rollingupdate strategy with partitions
	if instance.Spec.UpdateStrategy.RollingUpdate == nil ||
		instance.Spec.UpdateStrategy.RollingUpdate.Partition == nil {
		return reconcile.Result{}, nil
	}

	// Partition and replicas numbers defines our position in the rollout
	currentPartition := *instance.Spec.UpdateStrategy.RollingUpdate.Partition
	nReplicas := *instance.Spec.Replicas

	// If there's no ongoing rollouts, we're done
	if instance.Status.UpdateRevision == instance.Status.CurrentRevision {
		// Reset partition to max ordinal+1 and final hook call, if needed
		if currentPartition == 0 {
			return r.finishRollout(instance, hook)
		}
		return reconcile.Result{}, nil
	}

	// If we're at partition nReplicas, we're about to start a new rollout
	if currentPartition == nReplicas {
		return r.startRollout(instance, hook)
	}

	// Retrieve the pod for the current partition
	pod, err := r.getPod(instance, currentPartition)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Revision label is maintained by statefulset controller
	podRevision, ok := pod.GetLabels()[appsv1.StatefulSetRevisionLabel]
	if !ok {
		err := fmt.Errorf("pod missing %s label: %s",
			appsv1.StatefulSetRevisionLabel, pod.GetName())
		r.log.Error(err, "stsns", instance.GetNamespace(), "stsname", instance.GetName())
		return reconcile.Result{}, err
	}

	// This means the pod is up-to-date
	if podRevision == instance.Status.UpdateRevision && pod.Status.Phase == "Running" {
		// We're done
		if currentPartition == 0 {
			return r.finishRollout(instance, hook)
		}

		// Retrieve the next pod in line
		next, err := r.getPod(instance, currentPartition-1)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Ask hook if we should wait a bit longer before updating next pod
		if !hook.PodUpdateTransition(r.log, instance, pod, next) {
			return reconcile.Result{Requeue: true, RequeueAfter: retryInterval}, nil
		}

		// Pod is up-to-date and considered ready, let's resume rollout with the next pod
		return r.setPartitionNumber(instance, currentPartition-1)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileSts) startRollout(instance *appsv1.StatefulSet, hook hooks.STSRolloutHooks) (reconcile.Result, error) {
	nReplicas := *instance.Spec.Replicas
	pod, err := r.getPod(instance, nReplicas-1)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Ask hooks if we can start, or wait a bit longer
	if !hook.PodUpdateTransition(r.log, instance, nil, pod) {
		return reconcile.Result{Requeue: true, RequeueAfter: retryInterval}, nil
	}

	// Starts rollout with the higher pod number
	return r.setPartitionNumber(instance, nReplicas-1)
}

func (r *ReconcileSts) finishRollout(instance *appsv1.StatefulSet, hook hooks.STSRolloutHooks) (reconcile.Result, error) {
	nReplicas := *instance.Spec.Replicas
	pod, err := r.getPod(instance, 0)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !hook.PodUpdateTransition(r.log, instance, pod, nil) {
		return reconcile.Result{Requeue: true, RequeueAfter: retryInterval}, nil
	}

	// Reset partition number to max ordinal+1, so we'll intercept the next rollout
	instance.Spec.UpdateStrategy.RollingUpdate.Partition = &nReplicas
	err = r.Update(context.Background(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileSts) setPartitionNumber(instance *appsv1.StatefulSet, pos int32) (reconcile.Result, error) {
	r.log.Info("updating statefulset partition",
		"namespace", instance.GetNamespace(),
		"name", instance.GetName(),
		"partition", pos)

	instance.Spec.UpdateStrategy.RollingUpdate.Partition = &pos
	if err := r.Update(context.Background(), instance); err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: retryInterval}, nil
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSts) getPod(instance *appsv1.StatefulSet, pos int32) (*v1.Pod, error) {
	podName := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      fmt.Sprintf("%s-%d", instance.GetName(), pos),
	}
	pod := &v1.Pod{}
	err := r.Get(context.TODO(), podName, pod)
	if err != nil {
		return nil, err
	}
	return pod, nil
}
