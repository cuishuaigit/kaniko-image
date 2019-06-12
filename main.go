package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfig *string
	namespace  string
	repo       string
	project    string
	registry   string
	seq        []string
)

func main() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("c", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("c", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&repo, "repo", "", "provide a repo of your project")
	flag.StringVar(&namespace, "n", "default", "namespace")
	flag.StringVar(&project, "p", "", "your project name")
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
	canSplit := func(s rune) bool { return s == '/' }
	seq := strings.FieldsFunc(registry, canSplit)
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project,
			Namespace: namespace,
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
					Args:            []string{"--dockerfile=/workspace/" + project + "/Dockerfile", "--context=dir://workspace/" + project, "--destination=" + registry, "--cache=true", "--cache-repo=" + seq[0] + "/cache", "--cleanup"},
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
				"kaniko": "enabled",
			},
			Tolerations: []apiv1.Toleration{
				{
					Key:      "kaniko",
					Operator: apiv1.TolerationOperator(apiv1.TolerationOpEqual),
					Value:    "enabled",
					Effect:   apiv1.TaintEffect(apiv1.TaintEffectNoSchedule),
				},
			},
			RestartPolicy: apiv1.RestartPolicy(apiv1.RestartPolicyNever),
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
	}

	// Deletepods
	err = clientset.CoreV1().Pods(namespace).Delete(project, &metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("delete pod err :%v", err)
	}
	_, err = clientset.CoreV1().Pods(namespace).Create(pod)
	if err != nil {
		panic(err)
	}
}
