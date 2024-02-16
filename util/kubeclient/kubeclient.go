package kubeclient

import (
	"context"
	"fmt"

	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "neeraj.angi/app-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	AnnotationHash = fmt.Sprintf("%v/hash", v1.GroupVersion.Group)
)

func Apply(ctx context.Context, c client.Client, desired client.Object) error {
	existing := desired.DeepCopyObject().(client.Object)
	objKey := client.ObjectKeyFromObject(desired)
	err := c.Get(ctx, objKey, existing)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get %v: %v", objKey, err)
	}

	if errors.IsNotFound(err) {
		return c.Create(ctx, desired)
	}

	desiredHash, err := hashstructure.Hash(desired, hashstructure.FormatV2, &hashstructure.HashOptions{
		SlicesAsSets:    true,
		IgnoreZeroValue: true,
		ZeroNil:         true,
	})
	if err != nil {
		return fmt.Errorf("calculating hash: %v", err)
	}

	if existingHash, found := existing.GetAnnotations()[AnnotationHash]; !found || existingHash != fmt.Sprint(desiredHash) {
		desired.SetAnnotations(lo.Assign[string, string](
			existing.GetAnnotations(),
			map[string]string{AnnotationHash: fmt.Sprint(desiredHash)},
		))
		return c.Update(ctx, desired)
	}
	return nil
}
