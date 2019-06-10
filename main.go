package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfig *string
	namespace  string
	repo       string
	project    string
	registry   string
	seq         []string
)

func main() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("c", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("c", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&repo, "repo", "", "provide a repo of your project")
	flag.StringVar(&namespace, "n", "default", "namespace")
	flag.StringVar(&project, "p", "", "your projrct name")
	flag.StringVar(&registry, "r", "", "your registry name")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	deploymentClient := clientset.AppsV1().Deployments(namespace)
	if getDeployment(deploymentClient) == project {
		deleteDeployment(deploymentClient)
	}
	createDeployment(deploymentClient)
}

func int32Ptr(i int32) *int32 { return &i }
// CreateDeployment
func createDeployment(deploymentClient v1.DeploymentInterface) {
	canSplit := func(s rune) bool { return s == '/' }
	seq := strings.FieldsFunc(registry, canSplit)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kaniko",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kaniko",
					},
				},
				Spec: apiv1.PodSpec{
					InitContainers: []apiv1.Container{
						{
							Name:            "init-repo",
							Image:           "alpine/git",
							ImagePullPolicy: apiv1.PullPolicy(apiv1.PullIfNotPresent),
							Command:         []string{"git", "clone", repo},

							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "workdir",
									MountPath: "/git",
								},
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name:            "kaniko",
							Image:           "gcr.io/kaniko-project/executor:debug",
							ImagePullPolicy: apiv1.PullPolicy(apiv1.PullIfNotPresent),
							Args:            []string{"--dockerfile=/workspace/" + project + "/Dockerfile", "--context=dir://workspace/" + project, "--destination=" + registry,"--cache=true","--cache-repo="+seq[0]+"/cache","--cleanup"},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "workdir",
									MountPath: "/workspace",
								},
								{
									Name:      "kaniko-secret",
									MountPath: "/kaniko/.docker/",
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"kaniko":"enabled",
					} ,
					Tolerations: []apiv1.Toleration{
						{
							Key: "kaniko",
							Operator: apiv1.TolerationOperator(apiv1.TolerationOpEqual),
							Value: "enabled",
							Effect: apiv1.TaintEffect(apiv1.TaintEffectNoSchedule),
							},
					},
					Volumes: []apiv1.Volume{
						apiv1.Volume{
							Name: "kaniko-secret",
							VolumeSource: apiv1.VolumeSource{
								Secret: &apiv1.SecretVolumeSource{
									SecretName: "kaniko-secret",
								},
							},
						},
						apiv1.Volume{
							Name: "workdir",
						},
					},
				},
			},
		},
	}
	result, err := deploymentClient.Create(deployment)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Create deployment %q.\n", result.GetObjectMeta().GetName())
}

// DeleteDeployment
func deleteDeployment(deploymentClient v1.DeploymentInterface) {
	deletePolicy := metav1.DeletePropagationBackground
	if err := deploymentClient.Delete(project, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
}

// Getdeployment
func getDeployment(deploymentClient v1.DeploymentInterface) string {
	deployment, err := deploymentClient.Get(project, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	// fmt.Printf("%s\n", deployment.Name)
	return deployment.Name
}
