package cmd

import (
	"context"
	"errors"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"testing"
	"time"
)

type mockOC struct {
	runFunc         func(arg ...string) error
	runWithFileFunc func(outputFile string, arg ...string) error
}

func (o *mockOC) Run(arg ...string) error {
	if o.runFunc != nil {
		return o.runFunc(arg...)
	}
	return errors.New("method not implemented")
}

func (o *mockOC) RunWithOutputFile(outputFile string, arg ...string) error {
	if o.runWithFileFunc != nil {
		return o.runWithFileFunc(outputFile, arg...)
	}
	return errors.New("method not implemented")
}

func TestRun(t *testing.T) {
	client := fake.NewSimpleClientset()
	outputDir, err := ioutil.TempDir("/tmp", "run-tests-container-")
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	defer os.RemoveAll(outputDir)
	namespace := "test"
	tests := []*TestContainer{
		{
			Name:    "test1",
			Image:   "test1-image",
			Timeout: 2,
			Success: false,
		},
	}
	cmd := &runTestsCmd{
		clientset: client,
		tests:     tests,
		outputDir: outputDir,
		namespace: namespace,
		oc: &mockOC{
			runFunc: func(arg ...string) error {
				return nil
			},
			runWithFileFunc: func(outputFile string, arg ...string) error {
				return nil
			},
		},
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		pod, err := createPod(client, namespace)
		if err != nil {
			t.Fatalf("failed to create pod: %v", err)
		}

		time.Sleep(1 * time.Second)
		_, err = terminateContainer(client, namespace, pod)
		if err != nil {
			t.Errorf("failed to update pod due to error: %v", err)
		}
	}()

	err = cmd.run(context.TODO())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
		return
	}
	if !tests[0].Success {
		t.Fatalf("test didn't pass")
	}
}

func createPod(client kubernetes.Interface, namespace string) (*v1.Pod, error) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: namespace,
			Labels:    map[string]string{"job-name": "test1"},
		},
		Spec: v1.PodSpec{},
		Status: v1.PodStatus{
			Phase: v1.PodPending,
		},
	}
	return client.CoreV1().Pods(namespace).Create(pod)
}

func terminateContainer(client kubernetes.Interface, namespace string, pod *v1.Pod) (*v1.Pod, error) {
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
	return client.CoreV1().Pods(namespace).Update(updated)
}
