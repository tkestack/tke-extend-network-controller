package kube

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PatchMap(ctx context.Context, apiClient client.Client, obj client.Object, patchMap map[string]any) error {
	patch, err := json.Marshal(patchMap)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := apiClient.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch)); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
