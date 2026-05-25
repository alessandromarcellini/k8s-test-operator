/*
Copyright 2026.

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	demov1 "github.com/alessandromarcellini/k8s-test-operator/api/v1"
)

// ToyReconciler reconciles a Toy object
type ToyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=demo.test.com,resources=toys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=demo.test.com,resources=toys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=demo.test.com,resources=toys/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Toy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *ToyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	toy := &demov1.Toy{}

	if err := r.Get(ctx, req.NamespacedName, toy); err != nil { // its a kubectl get on the namespace of the request
		log.Error(err, "Failed to get Toy")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//log.Info("[RECONCILER] Reconciliation of Test resource complete!")
	log.Info("Toy reconciled",
		"replicas", toy.Spec.Replicas,
		"namespace", req.Namespace,
		"name", req.Name,
	)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ToyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.Toy{}).
		Named("toy").
		Complete(r)
}
