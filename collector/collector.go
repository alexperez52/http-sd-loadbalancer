package collector

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Get(ctx context.Context, LabelSelector map[string]string) ([]string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	collectors := []string{}
	// Namespace will be taken from env passed to the pod
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	for i := range pods.Items {
		collectors = append(collectors, pods.Items[i].Name)
	}

	// return collectors, nil
	// Returning dummy list for now
	return []string{"collector-1", "collector-2", "collector-3"}, nil
}
