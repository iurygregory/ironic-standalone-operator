package ironic

import (
	"fmt"
	"net"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metal3api "github.com/metal3-io/ironic-operator/api/v1alpha1"
)

const (
	// The name of the service for the database.
	DatabaseServiceName = "metal3-database"

	databaseAppName = "ironic-database"
	databasePort    = 3306
	databaseUser    = 27
)

func deploymentName(db *metal3api.IronicDatabase) string {
	return fmt.Sprintf("%s-database", db.Name)
}

func databasePasswordEnvVar(db *metal3api.IronicDatabase) corev1.EnvVar {
	return corev1.EnvVar{
		Name: "MARIADB_PASSWORD",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: db.Spec.CredentialsSecretName,
				},
				Key: "password",
			},
		},
	}

}

func newDatabasePodTemplate(db *metal3api.IronicDatabase) corev1.PodTemplateSpec {
	volumes := []corev1.Volume{}
	mounts := []corev1.VolumeMount{}

	if db.Spec.TLSSecretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "cert-mariadb",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: db.Spec.TLSSecretName,
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "cert-mariadb",
			MountPath: "/certs/mariadb",
		})
	}

	containers := []corev1.Container{
		{
			Name:  "mariadb",
			Image: db.Spec.Image,
			// TODO(dtantsur): livenessProbe+readinessProbe
			Env: []corev1.EnvVar{
				databasePasswordEnvVar(db),
				{
					Name:  "RESTART_CONTAINER_CERTIFICATE_UPDATED",
					Value: "true",
				},
			},
			VolumeMounts: mounts,
			SecurityContext: &corev1.SecurityContext{
				// FIXME(dtantsur): this should not be necessary, but our rootless approach does not always work:
				// https://github.com/metal3-io/mariadb-image/pull/8#issuecomment-1593004837
				Privileged:   pointer.BoolPtr(true),
				RunAsNonRoot: pointer.BoolPtr(false),
			},
		},
	}

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{metal3api.IronicOperatorLabel: databaseAppName},
		},
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	}
}

func ensureDatabaseDeployment(cctx ControllerContext, db *metal3api.IronicDatabase) (metal3api.IronicStatusConditionType, error) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: deploymentName(db), Namespace: db.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(cctx.Context, cctx.Client, deploy, func() error {
		if deploy.ObjectMeta.CreationTimestamp.IsZero() {
			cctx.Logger.Info("creating a new deployment")
			matchLabels := map[string]string{metal3api.IronicOperatorLabel: databaseAppName}
			deploy.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: matchLabels,
			}
		}
		deploy.Spec.Template = newDatabasePodTemplate(db)
		return controllerutil.SetControllerReference(db, deploy, cctx.Scheme)
	})
	if err != nil {
		return metal3api.IronicStatusProgressing, err
	}
	return getDeploymentStatus(deploy)
}

func buildEndpoints(ips []string) (endpoints []string) {
	port := fmt.Sprint(databasePort)
	for _, ip := range ips {
		endpoints = append(endpoints, net.JoinHostPort(ip, port))
	}
	sort.Strings(endpoints)
	return
}

func ensureDatabaseService(cctx ControllerContext, db *metal3api.IronicDatabase) (metal3api.IronicStatusConditionType, []string, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: deploymentName(db), Namespace: db.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(cctx.Context, cctx.Client, service, func() error {
		if service.ObjectMeta.Labels == nil {
			cctx.Logger.Info("creating a new service")
			service.ObjectMeta.Labels = make(map[string]string)
		}
		service.ObjectMeta.Labels[metal3api.IronicOperatorLabel] = databaseAppName

		service.Spec.Selector = map[string]string{metal3api.IronicOperatorLabel: databaseAppName}
		service.Spec.Ports = []corev1.ServicePort{
			{
				Protocol: corev1.ProtocolTCP,
				Port:     databasePort,
			},
		}
		service.Spec.Type = corev1.ServiceTypeClusterIP

		return controllerutil.SetControllerReference(db, service, cctx.Scheme)
	})
	if err != nil || len(service.Spec.ClusterIPs) == 0 {
		return metal3api.IronicStatusProgressing, nil, err
	}

	return metal3api.IronicStatusAvailable, buildEndpoints(service.Spec.ClusterIPs), nil
}

// EnsureDatabase ensures MariaDB is running with the current configuration.
func EnsureDatabase(cctx ControllerContext, db *metal3api.IronicDatabase) (status metal3api.IronicStatusConditionType, endpoints []string, err error) {
	status, err = ensureDatabaseDeployment(cctx, db)
	if err != nil || status != metal3api.IronicStatusAvailable {
		return
	}

	return ensureDatabaseService(cctx, db)
}

// RemoveDatabase removes the MariaDB database.
func RemoveDatabase(cctx ControllerContext, db *metal3api.IronicDatabase) error {
	return nil
}
