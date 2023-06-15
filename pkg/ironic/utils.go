package ironic

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metal3api "github.com/metal3-io/ironic-operator/api/v1alpha1"
)

type ControllerContext struct {
	Context    context.Context
	Client     client.Client
	KubeClient kubernetes.Interface
	Scheme     *runtime.Scheme
	Logger     logr.Logger
}

func getDeploymentStatus(deploy *appsv1.Deployment) (metal3api.IronicStatusConditionType, error) {
	if deploy.Status.ObservedGeneration != deploy.Generation {
		return metal3api.IronicStatusProgressing, nil
	}

	var available bool
	var err error
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
			available = true
		}
		if cond.Type == appsv1.DeploymentReplicaFailure && cond.Status == corev1.ConditionTrue {
			err = errors.Errorf("deployment failed: %s", cond.Message)
			return metal3api.IronicStatusProgressing, err
		}
	}

	if available {
		return metal3api.IronicStatusAvailable, nil
	} else {
		return metal3api.IronicStatusProgressing, nil
	}
}
