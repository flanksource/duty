package kubernetes

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var defaulter = runtime.NewScheme()

func Defaults(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if obj == nil {
		return nil, nil
	}

	if _, found, _ := unstructured.NestedString(obj.Object, "metadata", "creationTimestamp"); !found {
		unstructured.SetNestedField(obj.Object, nil, "metadata", "creationTimestamp")
	}

	switch obj.GetKind() {
	case "Deployment":
		deploy, err := AsDeployment(obj)
		if err != nil {
			return nil, err
		}
		return toUnstructured(obj, DefaultsDeployment(deploy))
	case "DaemonSet":
		daemonset, err := AsDaemonSet(obj)
		if err != nil {
			return nil, err
		}
		return toUnstructured(obj, DefaultsDaemonSet(daemonset))
	case "StatefulSet":
		sts, err := AsStatefulSet(obj)
		if err != nil {
			return nil, err
		}
		return toUnstructured(obj, DefaultsStatefulSet(sts))
	case "Service":
		svc, err := AsService(obj)
		if err != nil {
			return nil, err
		}
		return toUnstructured(obj, DefaultsService(svc))
	case "RoleBinding":
		rb, err := AsRoleBinding(obj)
		if err != nil {
			return nil, err
		}
		return toUnstructured(obj, DefaultsRoleBinding(rb))
	case "ClusterRoleBinding":
		rb, err := AsClusterRoleBinding(obj)
		if err != nil {
			return nil, err
		}
		return toUnstructured(obj, DefaultsClusterRoleBinding(rb))
	default:
		return obj, nil
	}
}

func AsClusterRoleBinding(obj *unstructured.Unstructured) (*rbac.ClusterRoleBinding, error) {
	if obj == nil {
		return nil, nil
	}

	var crb rbac.ClusterRoleBinding
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &crb); err != nil {
		return nil, err
	}
	return &crb, nil
}

func AsRoleBinding(obj *unstructured.Unstructured) (*rbac.RoleBinding, error) {
	if obj == nil {
		return nil, nil
	}

	var rb rbac.RoleBinding
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rb); err != nil {
		return nil, err
	}
	return &rb, nil
}

func AsService(obj *unstructured.Unstructured) (*v1.Service, error) {
	if obj == nil {
		return nil, nil
	}

	var svc v1.Service
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &svc); err != nil {
		return nil, err
	}
	return &svc, nil
}

func AsDaemonSet(obj *unstructured.Unstructured) (*appsv1.DaemonSet, error) {
	if obj == nil {
		return nil, nil
	}
	var daemonset appsv1.DaemonSet
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &daemonset); err != nil {
		return nil, err
	}
	return &daemonset, nil
}

func AsDeployment(obj *unstructured.Unstructured) (*appsv1.Deployment, error) {
	if obj == nil {
		return nil, nil
	}
	var deployment appsv1.Deployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

func AsStatefulSet(obj *unstructured.Unstructured) (*appsv1.StatefulSet, error) {
	if obj == nil {
		return nil, nil
	}
	var sts appsv1.StatefulSet
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sts); err != nil {
		return nil, err
	}
	return &sts, nil
}

func toUnstructured(unstructuredObj *unstructured.Unstructured, obj interface{}) (*unstructured.Unstructured, error) {
	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	unstructuredObj.Object = out
	return unstructuredObj, nil
}

func DefaultsDaemonSet(daemeonset *appsv1.DaemonSet) *appsv1.DaemonSet {
	defaulter.Default(daemeonset)
	daemeonset.Spec.RevisionHistoryLimit = defaultInt(daemeonset.Spec.RevisionHistoryLimit, 10)
	if daemeonset.Spec.UpdateStrategy.Type == "" {
		daemeonset.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: intStrPtr(1),
			},
		}
	}
	daemeonset.Spec.Template = DefaultsPod(daemeonset.Spec.Template)
	return daemeonset
}

func DefaultsStatefulSet(sts *appsv1.StatefulSet) *appsv1.StatefulSet {
	defaulter.Default(sts)
	sts.Spec.RevisionHistoryLimit = defaultInt(sts.Spec.RevisionHistoryLimit, 10)
	if sts.Spec.PodManagementPolicy == "" {
		sts.Spec.PodManagementPolicy = appsv1.OrderedReadyPodManagement
	}
	if sts.Spec.UpdateStrategy.Type == "" {
		sts.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
				Partition: intPtr(0),
			},
		}
	}
	sts.Spec.Template = DefaultsPod(sts.Spec.Template)
	return sts
}

func DefaultsDeployment(deploy *appsv1.Deployment) *appsv1.Deployment {
	defaulter.Default(deploy)
	deploy.Spec.ProgressDeadlineSeconds = defaultInt(deploy.Spec.ProgressDeadlineSeconds, 600)
	deploy.Spec.RevisionHistoryLimit = defaultInt(deploy.Spec.RevisionHistoryLimit, 10)
	if deploy.Spec.Strategy.Type == "" {
		deploy.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: intStrPtr("25%"),
				MaxSurge:       intStrPtr("25%"),
			},
		}
	}
	deploy.Spec.Template = DefaultsPod(deploy.Spec.Template)
	return deploy
}

func DefaultsClusterRoleBinding(rb *rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding {
	if rb == nil {
		return nil
	}
	defaulter.Default(rb)

	rb.Subjects = DefaultSubjects(rb.Subjects)
	return rb
}

func DefaultsService(svc *v1.Service) *v1.Service {
	defaulter.Default(svc)
	_ports := []v1.ServicePort{}
	for _, port := range svc.Spec.Ports {
		if port.Protocol == "" {
			port.Protocol = v1.ProtocolTCP
		}
		_ports = append(_ports, port)
	}
	svc.Spec.Ports = _ports
	return svc
}

func DefaultsRoleBinding(rb *rbac.RoleBinding) *rbac.RoleBinding {
	if rb == nil {
		return nil
	}
	defaulter.Default(rb)
	rb.Subjects = DefaultSubjects(rb.Subjects)
	return rb
}

func DefaultSubjects(subjects []rbac.Subject) []rbac.Subject {
	_subjects := []rbac.Subject{}
	for _, subject := range subjects {
		if subject.Kind == "ServiceAccount" {
			subject.APIGroup = ""
		}
		if subject.Kind == "User" {
			subject.APIGroup = "rbac.authorization.k8s.io"
		}
		_subjects = append(subjects, subject)
	}
	return _subjects
}

func DefaultsContainers(containers []v1.Container) []v1.Container {
	_containers := []v1.Container{}
	for _, container := range containers {
		_containers = append(_containers, DefaultsContainer(container))
	}
	return _containers
}

func DefaultsContainer(container v1.Container) v1.Container {
	if container.TerminationMessagePolicy == "" {
		container.TerminationMessagePolicy = v1.TerminationMessageReadFile
		container.TerminationMessagePath = v1.TerminationMessagePathDefault
	}
	_ports := []v1.ContainerPort{}
	for _, port := range container.Ports {
		if port.Protocol == "" {
			port.Protocol = v1.ProtocolTCP
		}
		_ports = append(_ports, port)
	}
	_env := []v1.EnvVar{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.FieldRef != nil && env.ValueFrom.FieldRef.APIVersion == "" {
			env.ValueFrom.FieldRef.APIVersion = "v1"
		}
		_env = append(_env, env)
	}
	container.Env = _env
	container.Ports = _ports
	if container.ImagePullPolicy == "" {
		container.ImagePullPolicy = v1.PullIfNotPresent
	}
	container.LivenessProbe = DefaultsProbe(container.LivenessProbe)
	container.ReadinessProbe = DefaultsProbe(container.ReadinessProbe)
	return container
}

func DefaultsProbe(probe *v1.Probe) *v1.Probe {
	if probe == nil {
		return nil
	}

	if probe.FailureThreshold == 0 {
		probe.FailureThreshold = 3
	}
	if probe.PeriodSeconds == 0 {
		probe.PeriodSeconds = 10
	}
	if probe.SuccessThreshold == 0 {
		probe.SuccessThreshold = 1
	}
	if probe.HTTPGet != nil && probe.HTTPGet.Scheme == "" {
		probe.HTTPGet.Scheme = v1.URISchemeHTTP
	}
	if probe.TimeoutSeconds == 0 {
		probe.TimeoutSeconds = 1
	}

	return probe
}

func DefaultsPod(pod v1.PodTemplateSpec) v1.PodTemplateSpec {
	pod.Spec.Containers = DefaultsContainers(pod.Spec.Containers)
	pod.Spec.InitContainers = DefaultsContainers(pod.Spec.InitContainers)
	volumes := []v1.Volume{}
	for _, volume := range pod.Spec.Volumes {
		readonly := int32(420)
		if volume.ConfigMap != nil && volume.ConfigMap.DefaultMode == nil {
			volume.ConfigMap.DefaultMode = &readonly
		}
		if volume.Secret != nil && volume.Secret.DefaultMode == nil {
			volume.Secret.DefaultMode = &readonly
		}
		volumes = append(volumes, volume)
	}
	pod.Spec.Volumes = volumes

	pod.Spec.TerminationGracePeriodSeconds = defaultInt64(pod.Spec.TerminationGracePeriodSeconds, 30)

	if pod.Spec.ServiceAccountName != "" && pod.Spec.ServiceAccountName != pod.Spec.DeprecatedServiceAccount {
		pod.Spec.DeprecatedServiceAccount = pod.Spec.ServiceAccountName
	}
	if pod.Spec.DeprecatedServiceAccount != "" && pod.Spec.ServiceAccountName != pod.Spec.DeprecatedServiceAccount {
		pod.Spec.ServiceAccountName = pod.Spec.DeprecatedServiceAccount
	}
	if pod.Spec.DNSPolicy == "" {
		pod.Spec.DNSPolicy = v1.DNSClusterFirst
	}
	if pod.Spec.RestartPolicy == "" {
		pod.Spec.RestartPolicy = v1.RestartPolicyAlways
	}
	if pod.Spec.SchedulerName == "" {
		pod.Spec.SchedulerName = "default-scheduler"
	}
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &v1.PodSecurityContext{}
	}
	return pod
}

func defaultInt64(on *int64, def int64) *int64 {
	if on != nil {
		return on
	}
	return &def
}

func defaultInt(on *int32, def int32) *int32 {
	if on != nil {
		return on
	}
	return &def
}

func intStrPtr(val any) *intstr.IntOrString {
	var ptr intstr.IntOrString
	switch v := val.(type) {
	case string:
		ptr = intstr.FromString(v)
	case int:
		ptr = intstr.FromInt(v)
	}
	return &ptr
}

func intPtr(i int32) *int32 { return &i }
