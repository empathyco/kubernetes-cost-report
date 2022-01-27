package pkg

import (
	"context"
	"fmt"
	"log"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

// NodeMetrics is a structure that contains the metrics for a node.
type NodeMetrics struct {
	Name        string  `json:"name"`
	MachineType string  `json:"machine_type"`
	CPUUsage    float64 `json:"cpu_usage"`
	Region      string  `json:"region"`
	MemoryUsage float64 `json:"memory_usage"`
}

// PodMetrics is a structure that contains a list of NodeMetrics.
type PodMetrics struct {
	Name        string  `json:"name"`
	Node        string  `json:"node"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
}

// Metrics is a structure that contains a list of PodMetrics and NodeMetrics.
type Metrics struct {
	Nodes      []NodeMetrics
	Pods       []PodMetrics
	Context    context.Context
	KubeConfig *string
}

// NewMetrics returns and inicialize a new Metrics structure.
func NewMetrics(context context.Context, kubeconfig *string) *Metrics {
	metrics := &Metrics{
		Context:    context,
		KubeConfig: kubeconfig,
	}
	if err := metrics.GetMetrics(); err != nil {
		log.Fatal(err)
	}

	return metrics
}

// GetMetrics returns the metrics for all nodes and pods.
func (m *Metrics) GetMetrics() error {
	config, err := clientcmd.BuildConfigFromFlags("", *m.KubeConfig)
	if err != nil {
		return fmt.Errorf("get metrics config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}
	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("get metrics: %w", err)
	}

	err = m.GetNodeMetrics(metricsClient, clientset)
	if err != nil {
		return err
	}

	return m.GetPodMetrics(metricsClient, clientset)
}

// GetNodeMetrics returns the metrics for all nodes.
func (m *Metrics) GetNodeMetrics(mc *metrics.Clientset, clientset *kubernetes.Clientset) error {
	metrics, err := mc.MetricsV1beta1().NodeMetricses().List(m.Context, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("node metrics: %w", err)
	}
	nodes, err := clientset.CoreV1().Nodes().List(m.Context, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("node info: %w", err)
	}
	m.Nodes = make([]NodeMetrics, 0)
	for nodeMetrics := range metrics.Items {
		node := findNode(nodes, metrics.Items[nodeMetrics].Name)
		if node != nil {
			m.Nodes = append(m.Nodes, NodeMetrics{
				Name:        node.Name,
				MachineType: node.Labels["beta.kubernetes.io/instance-type"],
				Region:      node.Labels["topology.kubernetes.io/region"],
				CPUUsage:    calcPercentageNodeCPU(node, &metrics.Items[nodeMetrics]),
				MemoryUsage: calcPercentageNodeMem(node, &metrics.Items[nodeMetrics]),
			})
		}
	}

	return nil
}

// GetPodMetrics returns the metrics for all pods.
func (m *Metrics) GetPodMetrics(mc *metrics.Clientset, clientset *kubernetes.Clientset) error {
	metrics, err := mc.MetricsV1beta1().PodMetricses(v1.NamespaceAll).List(m.Context, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("pod metrics: %w", err)
	}
	pods, err := clientset.CoreV1().Pods(v1.NamespaceAll).List(m.Context, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("pod info: %w", err)
	}
	m.Pods = make([]PodMetrics, 0)
	for podMetrics := range metrics.Items {
		pod := findPod(pods, metrics.Items[podMetrics].Name)
		if pod != nil {
			m.Pods = append(m.Pods, PodMetrics{
				Name:        pod.Name,
				Node:        pod.Spec.NodeName,
				CPUUsage:    calcPercentagePodCPU(&metrics.Items[podMetrics]),
				MemoryUsage: calcPercentagePodMem(&metrics.Items[podMetrics]),
			})
		}
	}

	return nil
}

func findNode(nodes *v1.NodeList, nodeName string) *v1.Node {
	for _, node := range nodes.Items {
		if node.Name == nodeName {
			return &node
		}
	}

	return nil
}

func findPod(pods *v1.PodList, podName string) *v1.Pod {
	for _, pod := range pods.Items {
		if pod.Name == podName {
			return &pod
		}
	}

	return nil
}

func calcPercentageNodeCPU(node *v1.Node, metrics *v1beta1.NodeMetrics) float64 {
	return (metrics.Usage.Cpu().AsApproximateFloat64() / node.Status.Capacity.Cpu().AsApproximateFloat64()) * 100
}

func calcPercentageNodeMem(node *v1.Node, metrics *v1beta1.NodeMetrics) float64 {
	return (metrics.Usage.Memory().AsApproximateFloat64() / node.Status.Capacity.Memory().AsApproximateFloat64()) * 100
}

func calcPercentagePodCPU(metrics *v1beta1.PodMetrics) float64 {
	total := 0.0
	for _, container := range metrics.Containers {
		total += container.Usage.Cpu().AsApproximateFloat64()
	}

	return (total) * 1000
}

func calcPercentagePodMem(metrics *v1beta1.PodMetrics) float64 {
	total := 0.0
	for _, container := range metrics.Containers {
		total += container.Usage.Memory().AsApproximateFloat64()
	}

	return (total)
}

// PrintNodes prints the node metrics.
func (m *Metrics) PrintNodes() string {
	result := "Node\t\t\t\t\t\tMachine Type\tRegion\t\tCPU Usage\tMemory Usage"
	for _, node := range m.Nodes {
		line := fmt.Sprintf("%s\t%s\t%s\t%f\t%f", node.Name, node.MachineType, node.Region, node.CPUUsage, node.MemoryUsage)
		result = strings.Join([]string{result, line}, "\n")
	}

	return result
}

// PrintPods print pods.
func (m *Metrics) PrintPods() string {
	result := "Pod\t\t\t\t\t\tCPU Usage\tMemory Usage"
	for _, pod := range m.Pods {
		line := fmt.Sprintf("%s\t\t%f\t%f", pod.Name, pod.CPUUsage, pod.MemoryUsage)
		result = strings.Join([]string{result, line}, "\n")
	}

	return result
}
