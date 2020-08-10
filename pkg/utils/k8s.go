package utils

import (
	"errors"
	"fmt"
	userv1typedclient "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// Create the given Job object. If a job with the same name exists, it will delete the existing job first before creating.
func CreateJob(clientset kubernetes.Interface, job *batchv1.Job) (*batchv1.Job, error) {
	api := clientset.BatchV1().Jobs(job.GetNamespace())
	j, err := api.Get(job.GetName(), metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	if j != nil && j.GetName() != "" {
		if err = api.Delete(j.GetName(), &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
	}
	return api.Create(job)
}

// Create the given namespace if it's not already exist
func CreateNamespace(client kubernetes.Interface, namespace string) (*v1.Namespace, error) {
	api := client.CoreV1().Namespaces()
	var n *v1.Namespace
	var err error
	if n, err = api.Get(namespace, metav1.GetOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	if n == nil || (n != nil && n.GetName() == "") {
		n = &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		if n, err = api.Create(n); err != nil {
			return nil, err
		}
	}
	return n, nil
}

// Create the ServiceAccount with the given serviceAccountName in the given namespace
func CreateServiceAccount(client kubernetes.Interface, namespace string, serviceAccountName string) (*v1.ServiceAccount, error) {
	api := client.CoreV1().ServiceAccounts(namespace)
	var s *v1.ServiceAccount
	var err error
	if s, err = api.Get(serviceAccountName, metav1.GetOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	if s == nil || (s != nil && s.GetName() == "") {
		s = &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		}
		if s, err = api.Create(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Bind the clusterRole with the given clusterRoleName to the given serviceAccount, and assign the owner to the ClusterRoleBinding object
func CreateClusterRoleBinding(client kubernetes.Interface, sa *v1.ServiceAccount, clusterRoleName string, owner metav1.OwnerReference) (*rbacv1.ClusterRoleBinding, error) {
	var err error
	api := client.RbacV1().ClusterRoleBindings()
	r := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-access-",
			OwnerReferences: []metav1.OwnerReference{
				owner,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      sa.GetName(),
				Namespace: sa.GetNamespace(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
	}
	if r, err = api.Create(r); err != nil {
		return nil, err
	}
	return r, nil
}

func AddUsersToGroup(client userv1typedclient.GroupsGetter, users []string, group string) error {
	g, err := client.Groups().Get(group, metav1.GetOptions{})
	if err != nil {
		return err
	}
	existingUsers := sets.NewString(g.Users...)
	for _, u := range users {
		if existingUsers.Has(u) {
			continue
		}
		g.Users = append(g.Users, u)
	}
	if _, err := client.Groups().Update(g); err != nil {
		return err
	}
	return nil
}

func WaitForContainerToComplete(client kubernetes.Interface, namespace string, podSelector string, containerName string, timeout time.Duration, logPrefix string) (*v1.ContainerStateTerminated, error) {
	var err error
	var watcher watch.Interface
	api := client.CoreV1().Pods(namespace)
	if watcher, err = api.Watch(metav1.ListOptions{LabelSelector: podSelector}); err != nil {
		return nil, err
	}
	for {
		select {
		case e := <-watcher.ResultChan():
			if e.Object == nil {
				continue
			}
			pod, ok := e.Object.(*v1.Pod)
			if !ok {
				continue
			}
			fmt.Println(fmt.Sprintf("[%s] Pod=%s State=%s", logPrefix, pod.GetName(), pod.Status.Phase))
			switch e.Type {
			case watch.Modified:
				if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
					if status, found := GetContainerStatus(pod.Status.ContainerStatuses, containerName); found {
						if status.State.Running != nil {
							fmt.Println(fmt.Sprintf("[%s] Pod=%s Container=%s ContainerStatus=%s", logPrefix, pod.GetName(), containerName, "Running"))
						}
						if status.State.Terminated != nil {
							fmt.Println(fmt.Sprintf("[%s] Pod=%s Container=%s ContainerStatus=%s", logPrefix, pod.GetName(), containerName, "Terminated"))
							watcher.Stop()
							return status.State.Terminated, nil
						}
					}
				}
				continue
			default:
				continue
			}
		case <-time.After(timeout):
			fmt.Println(fmt.Sprintf("[%s] Timed out when running pod with selector: %s", logPrefix, podSelector))
			watcher.Stop()
			return nil, errors.New("timeout")
		}
	}
}

func GetContainerStatus(statuses []v1.ContainerStatus, name string) (v1.ContainerStatus, bool) {
	for i := range statuses {
		if statuses[i].Name == name {
			return statuses[i], true
		}
	}
	return v1.ContainerStatus{}, false
}

func GetPods(client kubernetes.Interface, namespace string, podSelector string) (*v1.PodList, error) {
	api := client.CoreV1().Pods(namespace)
	return api.List(metav1.ListOptions{LabelSelector: podSelector})
}

func GetPod(client kubernetes.Interface, namespace string, name string) (*v1.Pod, error) {
	return client.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
}
