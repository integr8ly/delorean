package utils

import (
	userv1 "github.com/openshift/api/user/v1"
	fakeuser "github.com/openshift/client-go/user/clientset/versioned/fake"
	userv1typedclient "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"reflect"
	"testing"
	"time"
)

func TestCreateJob(t *testing.T) {
	cases := []struct {
		description   string
		clientFactory func() kubernetes.Interface
	}{
		{
			description: "test delete and add",
			clientFactory: func() kubernetes.Interface {
				existingJob := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "test",
					},
				}
				client := fake.NewSimpleClientset(existingJob)
				return client
			},
		},
		{
			description: "test add",
			clientFactory: func() kubernetes.Interface {
				return fake.NewSimpleClientset()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			client := c.clientFactory()
			newJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
					Labels:    map[string]string{"test": "test"},
				},
			}
			job, err := CreateJob(client, newJob)
			if err != nil {
				t.Fatalf("Unexpected error when create job: %v", err)
			}
			if !reflect.DeepEqual(job.Labels, newJob.Labels) {
				t.Fatal("Job created doesn't have the right labels")
			}
		})
	}
}

func TestCreateNamespace(t *testing.T) {
	cases := []struct {
		description   string
		clientFactory func() kubernetes.Interface
	}{
		{
			description: "test namespace exists",
			clientFactory: func() kubernetes.Interface {
				existing := &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}
				client := fake.NewSimpleClientset(existing)
				return client
			},
		},
		{
			description: "test namespace not exists",
			clientFactory: func() kubernetes.Interface {
				return fake.NewSimpleClientset()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			client := c.clientFactory()
			namespaceName := "test"
			namespace, err := CreateNamespace(client, namespaceName)
			if err != nil {
				t.Fatalf("Unexpected error when create namespace: %v", err)
			}
			if namespace.GetName() != namespaceName {
				t.Fatalf("Expected namespace name:%s, got:%s", namespaceName, namespace.GetName())
			}
		})
	}
}

func TestCreateServiceAccount(t *testing.T) {
	cases := []struct {
		description   string
		clientFactory func() kubernetes.Interface
	}{
		{
			description: "test sa exists",
			clientFactory: func() kubernetes.Interface {
				existing := &v1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				}
				client := fake.NewSimpleClientset(existing)
				return client
			},
		},
		{
			description: "test sa not exists",
			clientFactory: func() kubernetes.Interface {
				return fake.NewSimpleClientset()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			client := c.clientFactory()
			namespace := "test"
			saName := "test"
			sa, err := CreateServiceAccount(client, namespace, saName)
			if err != nil {
				t.Fatalf("Unexpected error when create ServiceAccount: %v", err)
			}
			if sa.GetName() != saName {
				t.Fatalf("Expected sa name:%s, got:%s", saName, sa.GetName())
			}
		})
	}
}

func TestCreateClusterRoleBinding(t *testing.T) {
	cases := []struct {
		description string
		sa          *v1.ServiceAccount
		owner       metav1.OwnerReference
	}{
		{
			description: "create role binding",
			sa: &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-sa",
				},
			},
			owner: metav1.OwnerReference{
				APIVersion:         "v1",
				Kind:               "namespace",
				Name:               "test",
				Controller:         nil,
				BlockOwnerDeletion: nil,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			clusterRoleName := "cluster-admin"
			client := fake.NewSimpleClientset()
			roleBinding, err := CreateClusterRoleBinding(client, c.sa, clusterRoleName, c.owner)
			if err != nil {
				t.Fatalf("unexpected error when create rolebinding: %v", err)
			}
			if !reflect.DeepEqual(roleBinding.OwnerReferences[0], c.owner) {
				t.Fatalf("owner referece doesn't match. expected: %v, got: %v", c.owner, roleBinding.OwnerReferences[0])
			}
			if roleBinding.Subjects[0].Name != c.sa.GetName() {
				t.Fatalf("subject name doesn't match. expected: %s, got: %s", c.sa.GetName(), roleBinding.Subjects[0].Name)
			}
			if roleBinding.Subjects[0].Namespace != c.sa.GetNamespace() {
				t.Fatalf("subject namespace doesn't match. expected: %s, got: %s", c.sa.GetNamespace(), roleBinding.Subjects[0].Namespace)
			}
			if roleBinding.RoleRef.Name != clusterRoleName {
				t.Fatalf("roleRef name doesn't match. expected: %s, got: %s", clusterRoleName, roleBinding.RoleRef.Name)
			}
		})
	}
}

func TestAddUsersToGroup(t *testing.T) {
	cases := []struct {
		description string
		groupClient func() userv1typedclient.GroupsGetter
		group       string
	}{
		{
			description: "should add new user",
			groupClient: func() userv1typedclient.GroupsGetter {
				g := &userv1.Group{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-group",
					},
					Users: []string{"user1"},
				}
				return fakeuser.NewSimpleClientset(g).UserV1()
			},
			group: "test-group",
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			users := []string{"user1", "user2"}
			client := c.groupClient()
			err := AddUsersToGroup(client, users, c.group)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			g, _ := client.Groups().Get(c.group, metav1.GetOptions{})
			existingUsers := sets.NewString(g.Users...)
			for _, u := range users {
				if !existingUsers.Has(u) {
					t.Fatalf("user %s doesn't exist in group %s", u, g.Name)
				}
			}
		})
	}
}

func TestWaitForContainerToComplete(t *testing.T) {
	namespace := "test"
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test",
			Labels:    map[string]string{"job-name": "test"},
		},
		Spec: v1.PodSpec{},
		Status: v1.PodStatus{
			Phase: v1.PodPending,
		},
	}
	selector := "job-name=test"
	client := fake.NewSimpleClientset(pod)
	timeout := 3 * time.Second

	cases := []struct {
		description   string
		updateFunc    func()
		shouldTimeout bool
	}{
		{
			description: "should receive result",
			updateFunc: func() {
				// Need to wait for 1 second here otherwise the watcher is not set up when the object is updated
				time.Sleep(1 * time.Second)
				updated := pod.DeepCopy()
				updated.Status.Phase = v1.PodRunning
				updated.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						Name: "test",
						State: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				}
				_, err := client.CoreV1().Pods(namespace).Update(updated)
				if err != nil {
					t.Errorf("failed to update pod due to error: %v", err)
				}
			},
			shouldTimeout: false,
		},
		{
			description:   "should timeout",
			updateFunc:    func() {},
			shouldTimeout: true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			results := make(chan *v1.ContainerStateTerminated, 1)
			go func() {
				r, err := WaitForContainerToComplete(client, namespace, selector, "test", timeout, "test")
				if err != nil {
					t.Logf("unexpected error: %v", err)
					return
				}
				results <- r
			}()

			c.updateFunc()

			select {
			case r := <-results:
				t.Logf("Got result. Exit code: %d", r.ExitCode)
				if c.shouldTimeout {
					t.Fatal("should timed out, but got result")
				}
			case <-time.After(timeout):
				if !c.shouldTimeout {
					t.Fatal("No result received")
				} else {
					t.Log("got timeout as expected")
				}
			}
		})
	}
}
