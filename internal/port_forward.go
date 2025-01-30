package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds the main structure of the TOML configuration
type Config struct {
	GlobalKubeConfig string    `toml:"global_kubeconfig,omitempty"`
	Contexts         []Context `toml:"context"`
}

// Context holds Kubernetes context settings
type Context struct {
	Name           string     `toml:"name"`
	Address        string     `toml:"address"`
	Namespace      string     `toml:"namespace"`
	KubeConfigPath string     `toml:"kubeconfig,omitempty"`
	Svc            []Service  `toml:"svc"`
	Pods           []Pod      `toml:"pods"`
	LabelSelectors []Selector `toml:"label-selectors"`
}

// Service represents a Kubernetes service to be forwarded
type Service struct {
	Name      string    `toml:"name"`
	Ports     []PortMap `toml:"ports"`
	Namespace string    `toml:"namespace,omitempty"`
}

// Pod represents a Kubernetes pod to be forwarded
type Pod struct {
	Name      string    `toml:"name"`
	Ports     []PortMap `toml:"ports"`
	Namespace string    `toml:"namespace,omitempty"`
}

// Selector represents a label selector for forwarding
type Selector struct {
	Label     string    `toml:"label"`
	Ports     []PortMap `toml:"ports"`
	Namespace string    `toml:"namespace,omitempty"`
}

// PortMap represents a port-forward mapping (source -> target)
type PortMap struct {
	Source string `toml:"source"`
	Target string `toml:"target"`
}

func Portforward(ctx *Context, config *Config) {
	logrus.Infof("Processing context: %s\n", ctx.Name)

	// Ensure default namespace is set
	if ctx.Namespace == "" {
		ctx.Namespace = "default"
	}

	// Load Kubernetes client
	clientset, _, err := getKubeClient(ctx.KubeConfigPath, config.GlobalKubeConfig)
	if err != nil {
		logrus.Fatalf("Failed to load KubeClient: %v", err)
	}

	// Process Services
	for _, svc := range ctx.Svc {
		go func(service Service) {
			// Use service namespace if defined, otherwise fall back to context namespace
			namespace := service.Namespace
			if namespace == "" {
				namespace = ctx.Namespace
			}

			err := portForwardResource(ctx.Name, namespace, "svc/"+service.Name, svc.Ports, ctx.Address)
			if err != nil {
				logrus.Errorf("Error forwarding service %s: %v", service.Name, err)
			}
		}(svc)
	}

	// Process Pods
	for _, pod := range ctx.Pods {
		go func(pod Pod) {
			namespace := pod.Namespace
			if namespace == "" {
				namespace = ctx.Namespace
			}
			err := portForwardResource(ctx.Name, namespace, "pod/"+pod.Name, pod.Ports, ctx.Address)
			if err != nil {
				logrus.Errorf("Error forwarding pod %s: %v", pod.Name, err)
			}
		}(pod)
	}

	// Process Label Selectors
	for _, selector := range ctx.LabelSelectors {
		go func(sel Selector) {
			namespace := selector.Namespace
			if namespace == "" {
				namespace = ctx.Namespace
			}
			err := portForwardLabel(clientset, ctx.Name, namespace, sel.Label, sel.Ports, ctx.Address)
			if err != nil {
				logrus.Errorf("Error forwarding label selector %s: %v", sel.Label, err)
			}
		}(selector)
	}
}

// getKubeClient initializes a Kubernetes client
func getKubeClient(contextKubeConfig, globalKubeConfig string) (*kubernetes.Clientset, *rest.Config, error) {
	var config *rest.Config
	var err error

	// Use per-context kubeconfig if available, else global kubeconfig
	if contextKubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", contextKubeConfig)
	} else if globalKubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", globalKubeConfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return clientset, config, nil
}

// portForwardResource calls `kubectl port-forward` directly to support address binding.
func portForwardResource(context, namespace, resource string, ports []PortMap, address string) error {
	portMappings := []string{}
	for _, p := range ports {
		portMappings = append(portMappings, fmt.Sprintf("%s:%s", p.Source, p.Target))
	}

	cmdArgs := []string{"--context", context, "port-forward", resource}
	cmdArgs = append(cmdArgs, "--namespace", namespace)
	cmdArgs = append(cmdArgs, portMappings...)
	cmdArgs = append(cmdArgs, "--address", address)

	cmd := exec.Command("kubectl", cmdArgs...)
	cmd.Stdout = nil // Redirect output if needed
	cmd.Stderr = nil // Redirect error if needed

	go func() {
		err := cmd.Run()
		if err != nil {
			logrus.Errorf("Error executing port-forward for %s: %v", resource, err)
		}
	}()

	logrus.Infof("Started port-forward: kubectl %s", strings.Join(cmdArgs, " "))
	return nil
}

// portForwardLabel selects pods based on labels and starts port forwarding
func portForwardLabel(clientset *kubernetes.Clientset, ctx, namespace, label string, ports []PortMap, address string) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found with label: %s", label)
	}

	// Forward the first matching pod
	firstPod := pods.Items[0]
	return portForwardResource(ctx, namespace, firstPod.Name, ports, address)
}

