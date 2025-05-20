package velero

import (
	"context"
	"reflect"

	veleroInstall "github.com/vmware-tanzu/velero/pkg/install"

	"github.com/go-logr/logr"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InstallVeleroCRDs ensures that operator dependencies are installed at runtime.
func InstallVeleroCRDs(log logr.Logger, client client.Client) error {
	var err error

	// Install CRDs
	for _, unstructuredCrd := range veleroInstall.AllCRDs().Items {
		// Get upstream crds
		crd := &apiv1.CustomResourceDefinition{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCrd.Object, crd); err != nil {
			return err
		}

		// Add Conversion to the spec, as this will be returned in the founcCrd
		crd.Spec.Conversion = &apiv1.CustomResourceConversion{
			Strategy: apiv1.NoneConverter,
		}

		// Lookup for installed/pre-existing crds
		foundCrd := &apiv1.CustomResourceDefinition{}
		if err = client.Get(context.TODO(), types.NamespacedName{Name: crd.Name}, foundCrd); err != nil {
			if errors.IsNotFound(err) {
				// Didn't find CRD, we should create it.
				log.Info("Creating CRD", "CRD.Name", crd.Name)
				if err = client.Create(context.TODO(), crd); err != nil {
					return err
				}
			} else {
				// Return other errors
				return err
			}
		} else {
			// CRD exists, check if it's updated.
			if !reflect.DeepEqual(foundCrd.Spec, crd.Spec) {
				// Specs aren't equal, update and fix.
				log.Info("Updating CRD", "CRD.Name", crd.Name, "foundCrd.Spec", foundCrd.Spec, "crd.Spec", crd.Spec)
				foundCrd.Spec = *crd.Spec.DeepCopy()
				if err = client.Update(context.TODO(), foundCrd); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
