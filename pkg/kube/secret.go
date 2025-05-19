package kube

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCertIdFromSecret(ctx context.Context, apiClient client.Client, key client.ObjectKey) (string, error) {
	secret := &corev1.Secret{}
	if err := apiClient.Get(ctx, key, secret); err != nil {
		return "", errors.WithStack(err)
	}
	certId := secret.Data["qcloud_cert_id"]
	if len(certId) > 0 {
		return string(certId), nil
	}
	return "", nil
}
