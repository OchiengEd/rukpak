/*
Copyright 2020 The Operator-SDK Authors.

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

package predicate

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	crtpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("predicate")

type GenerationChangedPredicate = crtpredicate.GenerationChangedPredicate

// DependentPredicateFuncs returns functions defined for filtering events
func DependentPredicateFuncs[T client.Object]() crtpredicate.TypedFuncs[T] {
	dependentPredicate := crtpredicate.TypedFuncs[T]{
		// We don't need to reconcile dependent resource creation events
		// because dependent resources are only ever created during
		// reconciliation. Another reconcile would be redundant.
		CreateFunc: func(e event.TypedCreateEvent[T]) bool {
			o := e.Object
			log.V(1).Info("Skipping reconciliation for dependent resource creation", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GetObjectKind().GroupVersionKind().GroupVersion(), "kind", o.GetObjectKind().GroupVersionKind().Kind)
			return false
		},
		// Reconcile when a dependent resource is deleted so that it can be
		// recreated.
		DeleteFunc: func(e event.TypedDeleteEvent[T]) bool {
			o := e.Object
			log.V(1).Info("Reconciling due to dependent resource deletion", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GetObjectKind().GroupVersionKind().GroupVersion(), "kind", o.GetObjectKind().GroupVersionKind().Kind)
			return true
		},

		// Don't reconcile when a generic event is received for a dependent
		GenericFunc: func(e event.TypedGenericEvent[T]) bool {
			o := e.Object
			log.V(1).Info("Skipping reconcile due to generic event", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GetObjectKind().GroupVersionKind().GroupVersion(), "kind", o.GetObjectKind().GroupVersionKind().Kind)
			return false
		},

		// Reconcile when a dependent resource is updated, so that it can
		// be patched back to the resource managed by the CR, if
		// necessary. Ignore updates that only change the status and
		// resourceVersion.
		UpdateFunc: func(e event.TypedUpdateEvent[T]) bool {
			oldObj := e.ObjectOld.DeepCopyObject().(*unstructured.Unstructured)
			newObj := e.ObjectNew.DeepCopyObject().(*unstructured.Unstructured)

			delete(oldObj.Object, "status")
			delete(newObj.Object, "status")
			oldObj.SetResourceVersion("")
			newObj.SetResourceVersion("")

			if reflect.DeepEqual(oldObj.Object, newObj.Object) {
				return false
			}
			log.V(1).Info("Reconciling due to dependent resource update", "name", newObj.GetName(), "namespace", newObj.GetNamespace(), "apiVersion", newObj.GroupVersionKind().GroupVersion(), "kind", newObj.GroupVersionKind().Kind)
			return true
		},
	}

	return dependentPredicate
}
