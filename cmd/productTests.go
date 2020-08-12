package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
)

const (
	testJobParallelism  = 1
	testJobCompletions  = 1
	testJobBackoffLimit = 0
	defaultNamespace    = "rhmi-product-tests"
	serviceAccountName  = "cluster-admin-sa"
	secretName          = "rh-integration-auth"
)

type TestContainer struct {
	Name    string      `json:"name"`
	Image   string      `json:"image"`
	Timeout int64       `json:"timeout,omitempty"`
	EnvVars []v1.EnvVar `json:"envVars,omitempty"`
	Success bool
}

type testContainerList struct {
	Tests []*TestContainer `json:"tests"`
}

type runTestsCmdFlags struct {
	outputDir       string
	timout          int64
	namespace       string
	testsConfigFile string
}

type runTestsCmd struct {
	clientset kubernetes.Interface
	tests     []*TestContainer
	outputDir string
	namespace string
	oc        utils.OCInterface
}

func init() {
	f := &runTestsCmdFlags{}
	cmd := &cobra.Command{
		Use:   "product-tests",
		Short: "Execute RHMI product test containers",
		Run: func(cmd *cobra.Command, args []string) {
			kubeConfig, err := requireValue(KubeConfigKey)
			if err != nil {
				handleError(err)
			}
			c, err := newRunTestsCmd(kubeConfig, f)
			if err != nil {
				handleError(err)
			}
			if err = c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}

	pipelineCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.outputDir, "output", "o", "", "Absolute path of the output directory to save reports")
	cmd.MarkFlagRequired("output")
	cmd.Flags().StringVarP(&f.namespace, "namespace", "n", defaultNamespace, "The namespace to run the test containers")
	cmd.Flags().StringVar(&f.testsConfigFile, "test-config", "", "Path to the tests configuration file")
	cmd.MarkFlagRequired("test-config")
}

func newRunTestsCmd(kubeconfig string, f *runTestsCmdFlags) (*runTestsCmd, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dt := time.Now()
	outputDir := path.Join(f.outputDir, dt.Format("2006-01-02-03-04-05"))
	if err = os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, err
	}
	testList := &testContainerList{}
	if err := utils.PopulateObjectFromYAML(f.testsConfigFile, testList); err != nil {
		return nil, err
	}

	return &runTestsCmd{
		clientset: clientset,
		tests:     testList.Tests,
		outputDir: outputDir,
		namespace: f.namespace,
		oc:        utils.NewOC(kubeconfig),
	}, nil
}

func (c *runTestsCmd) run(ctx context.Context) error {
	var ns *v1.Namespace
	var sa *v1.ServiceAccount
	var err error
	username, err := requireValue(RHIntegrationUsername)
	if err != nil {
		return err
	}
	password, err := requireValue(RHIntegrationPassword)
	if err != nil {
		return err
	}
	fmt.Println("[Prepare] Create namespace", c.namespace)
	if ns, err = utils.CreateNamespace(c.clientset, c.namespace); err != nil {
		return err
	}
	fmt.Println("[Prepare] Create serviceAccount", serviceAccountName)
	if sa, err = utils.CreateServiceAccount(c.clientset, c.namespace, serviceAccountName); err != nil {
		return err
	}
	fmt.Println("[Prepare] Create ClusterRoleBinding for the service account")
	gvk := schema.FromAPIVersionAndKind("v1", "namespace")
	owner := metav1.NewControllerRef(ns, gvk)
	if _, err = utils.CreateClusterRoleBinding(c.clientset, sa, "cluster-admin", *owner); err != nil {
		return err
	}
	err = utils.CreateSecret(c.clientset, secretName, "quay.io", c.namespace, username, password)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, testContainer := range c.tests {
		wg.Add(1)
		go func(t *TestContainer) {
			defer wg.Done()
			fmt.Println(fmt.Sprintf("[%s] Start test container", t.Name))
			ok, err := c.runTestContainer(ctx, t)
			if err != nil {
				fmt.Println(fmt.Sprintf("[%s] Error when run test container: %v", t.Name, err))
			}
			if ok {
				testContainer.Success = true
				fmt.Println(fmt.Sprintf("[%s] Test container finished successfully", t.Name))
			} else {
				testContainer.Success = false
				fmt.Println(fmt.Sprintf("[%s] Test container failed", t.Name))
			}
		}(testContainer)
	}
	wg.Wait()
	fmt.Println(fmt.Sprintf("[Reporting] Tests completed. Results can be found in %s", c.outputDir))
	fmt.Println("[TearDown] Delete namespace", c.namespace)
	err = c.clientset.CoreV1().Namespaces().Delete(c.namespace, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *runTestsCmd) runTestContainer(ctx context.Context, test *TestContainer) (bool, error) {
	job := getTestContainerJob(c.namespace, test)
	if _, err := utils.CreateJob(c.clientset, job); err != nil {
		return false, err
	}
	podSelector := fmt.Sprintf("job-name=%s", job.GetName())
	var podList *v1.PodList
	var err error
	fmt.Println(fmt.Sprintf("[%s] Waiting for job to be started", test.Name))
	err = wait.PollImmediate(time.Duration(1)*time.Second, time.Duration(60)*time.Second, func() (done bool, err error) {
		if podList, err = utils.GetPods(c.clientset, c.namespace, podSelector); err != nil {
			return false, err
		}
		if len(podList.Items) == 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return false, errors.New(fmt.Sprintf("[%s] Failed to list pods for job %s", test.Name, job.GetName()))
	}
	pod := podList.Items[0]
	fmt.Println(fmt.Sprintf("[%s] Pod found for job: %s", test.Name, pod.GetName()))
	fmt.Println(fmt.Sprintf("[%s] Wait for test container to finish", test.Name))
	var containerResult *v1.ContainerStateTerminated
	timeout := time.Duration(test.Timeout) * time.Second
	if containerResult, err = utils.WaitForContainerToComplete(c.clientset, c.namespace, podSelector, "test", timeout, test.Name); err != nil {
		return false, err
	}
	fmt.Println(fmt.Sprintf("[%s] Tests completed. Exit code = %d", test.Name, containerResult.ExitCode))
	fmt.Println(fmt.Sprintf("[%s] Save test pod status", test.Name))
	if err = c.savePodStatus(pod, test.Name); err != nil {
		fmt.Println(fmt.Sprintf("[%s] Failed to save test pod status due to error: %v", test.Name, err))
	}
	fmt.Println(fmt.Sprintf("[%s] Download test results", test.Name))
	if err = c.downloadTestResults(pod, test.Name); err != nil {
		fmt.Println(fmt.Sprintf("[%s] Failed to download test result due to error: %v", test.Name, err))
	}
	fmt.Println(fmt.Sprintf("[%s] Download test container logs", test.Name))
	if err = c.downloadLogs(pod, test.Name); err != nil {
		fmt.Println(fmt.Sprintf("[%s] Failed to container logs due to error: %v", test.Name, err))
	}
	if err = c.completeJob(pod); err != nil {
		return false, err
	}
	fmt.Println(fmt.Sprintf("[%s] Delete test job", test.Name))
	err = c.clientset.BatchV1().Jobs(job.GetNamespace()).Delete(job.GetName(), &metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}

	return containerResult.ExitCode == 0, err
}

func getTestContainerJob(namespace string, t *TestContainer) *batchv1.Job {

	// extend the job timeout compared to the test timeout to allow the delorean
	// cli to retrieve the logs from the containers before the pod is destroyed
	// from the job
	var extendedTimeout = t.Timeout + 180
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism:           pointer.Int32Ptr(testJobParallelism),
			Completions:           pointer.Int32Ptr(testJobCompletions),
			ActiveDeadlineSeconds: &extendedTimeout,
			BackoffLimit:          pointer.Int32Ptr(testJobBackoffLimit),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "test-run-results",
						},
					},
					Containers: []v1.Container{
						{
							Name:  "test",
							Image: t.Image,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "test-run-results",
									MountPath: "/test-run-results",
								},
							},
							Env: t.EnvVars,
						},
						{
							Name:  "sidecar",
							Image: "busybox",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "test-run-results",
									MountPath: "/test-run-results",
								},
							},
							Command: []string{"sh"},
							Args:    []string{"-c", "while true; if [[ -f /tmp/done ]]; then exit 0; fi; do sleep 1; done"},
						},
					},
					RestartPolicy:      "Never",
					ServiceAccountName: serviceAccountName,
					ImagePullSecrets: []v1.LocalObjectReference{{
						Name: secretName,
					},
					},
				},
			},
		},
	}
}

func (c *runTestsCmd) savePodStatus(pod v1.Pod, testName string) error {
	p, err := utils.GetPod(c.clientset, c.namespace, pod.GetName())
	if err != nil {
		return err
	}
	o := path.Join(c.outputDir, testName, "logs", "pod.yaml")
	if err := os.MkdirAll(path.Dir(o), os.ModePerm); err != nil {
		return err
	}
	return utils.WriteObjectToYAML(p, o)
}

func (c *runTestsCmd) downloadTestResults(pod v1.Pod, testName string) error {
	from := fmt.Sprintf("%s:/test-run-results/", pod.GetName())
	to := path.Join(c.outputDir, testName, "results")
	if err := os.MkdirAll(to, os.ModePerm); err != nil {
		return err
	}
	return c.oc.Run("rsync", from, to, "-c", "sidecar", "-n", c.namespace)
}

func (c *runTestsCmd) downloadLogs(pod v1.Pod, testName string) error {
	outputFile := path.Join(c.outputDir, testName, "logs", "container.log")
	if err := os.MkdirAll(path.Dir(outputFile), os.ModePerm); err != nil {
		return err
	}
	return c.oc.RunWithOutputFile(outputFile, "logs", pod.GetName(), "-c", "test", "-n", c.namespace)
}

func (c *runTestsCmd) completeJob(pod v1.Pod) error {
	return c.oc.Run("exec", pod.GetName(), "-c", "sidecar", "-n", c.namespace, "--", "touch", "/tmp/done")
}
