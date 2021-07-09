package collector

import (
	"context"
)

func Get(ctx context.Context, LabelSelector map[string]string) ([]string, error) {
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	return nil, err
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	return nil, err
	// }

	// collectors := []string{}
	// pods, err := clientset.CoreV1().Pods(os.Getenv("OTEL_NAMESPACE")).List(context.TODO(), metav1.ListOptions{})
	// for i := range pods.Items {
	// 	collectors = append(collectors, pods.Items[i].Name)
	// }

	// return collectors, nil
	// Returning dummy list for now
	return []string{"collector-1", "collector-2", "collector-3"}, nil
}
