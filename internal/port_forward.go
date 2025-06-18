package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/logrusorgru/aurora/v4"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// Config holds the main structure of the TOML configuration
type Config struct {
	GlobalKubeConfig string    `toml:"global_kubeconfig,omitempty"`
	DefaultAddress   string    `toml:"default_address,omitempty"`
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
	Address   string    `toml:"address,omitempty"`
}

// Pod represents a Kubernetes pod to be forwarded
type Pod struct {
	Name      string    `toml:"name"`
	Ports     []PortMap `toml:"ports"`
	Namespace string    `toml:"namespace,omitempty"`
	Address   string    `toml:"address,omitempty"`
}

// Selector represents a label selector for forwarding
type Selector struct {
	Label     string    `toml:"label"`
	Ports     []PortMap `toml:"ports"`
	Namespace string    `toml:"namespace,omitempty"`
	Address   string    `toml:"address,omitempty"`
}

// PortMap represents a port-forward mapping (source -> target)
type PortMap struct {
	Source string `toml:"source"`
	Target string `toml:"target"`
}

func computeAddress(entryAddr, ctxAddr, globalAddr string) string {
	if entryAddr != "" {
		return entryAddr
	}
	if ctxAddr != "" {
		return ctxAddr
	}
	if globalAddr != "" {
		return globalAddr
	}
	return "0.0.0.0"
}

func Portforward(ctx *Context, config *Config) {
	logrus.Infof("%s: %s", aurora.Yellow("Processing context"), aurora.Bold(aurora.Cyan(ctx.Name)))

	if ctx.Namespace == "" {
		ctx.Namespace = "default"
	}

	clientset, cfg, err := getKubeClient(ctx.Name, ctx.KubeConfigPath, config.GlobalKubeConfig)
	if err != nil {
		logrus.Fatalf("Failed to load KubeClient: %v", err)
	}

	for _, svc := range ctx.Svc {
		go func(service Service) {
			namespace := service.Namespace
			if namespace == "" {
				namespace = ctx.Namespace
			}
			addr := computeAddress(service.Address, ctx.Address, config.DefaultAddress)
			err := portForwardResource(clientset, cfg, ctx.Name, namespace, "svc/"+service.Name, service.Ports, addr)
			if err != nil {
				logrus.Errorf("Error forwarding service %s: %v", service.Name, err)
			}
		}(svc)
	}

	for _, pod := range ctx.Pods {
		go func(pod Pod) {
			namespace := pod.Namespace
			if namespace == "" {
				namespace = ctx.Namespace
			}
			addr := computeAddress(pod.Address, ctx.Address, config.DefaultAddress)
			err := portForwardResource(clientset, cfg, ctx.Name, namespace, "pod/"+pod.Name, pod.Ports, addr)
			if err != nil {
				logrus.Errorf("Error forwarding pod %s: %v", pod.Name, err)
			}
		}(pod)
	}

	for _, selector := range ctx.LabelSelectors {
		go func(sel Selector) {
			namespace := sel.Namespace
			if namespace == "" {
				namespace = ctx.Namespace
			}
			addr := computeAddress(sel.Address, ctx.Address, config.DefaultAddress)
			err := portForwardLabel(clientset, cfg, ctx.Name, namespace, sel.Label, sel.Ports, addr)
			if err != nil {
				logrus.Errorf("Error forwarding label selector %s: %v", sel.Label, err)
			}
		}(selector)
	}
}

// getKubeClient initializes a Kubernetes client
func getKubeClient(contextName, contextKubeConfig, globalKubeConfig string) (*kubernetes.Clientset, *rest.Config, error) {
	var config *rest.Config
	var err error

	if contextKubeConfig != "" {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: contextKubeConfig}
		overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	} else if globalKubeConfig != "" {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: globalKubeConfig}
		overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
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

func portForwardResource(clientset *kubernetes.Clientset, cfg *rest.Config, contextName, namespace, resource string, ports []PortMap, address string) error {
	var podName string
	if strings.HasPrefix(resource, "svc/") {
		name := strings.TrimPrefix(resource, "svc/")
		svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get service %s: %v", name, err)
		}
		if len(svc.Spec.Selector) == 0 {
			return fmt.Errorf("service %s has no selector", name)
		}
		selector := labels.Set(svc.Spec.Selector).String()
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return fmt.Errorf("failed to list pods for service %s: %v", name, err)
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found for service %s", name)
		}
		podName = pods.Items[0].Name
	} else {
		podName = strings.TrimPrefix(resource, "pod/")
	}

	go maintainPortForward(cfg, contextName, namespace, podName, ports, address)
	return nil
}

func portForwardLabel(clientset *kubernetes.Clientset, cfg *rest.Config, contextName, namespace, label string, ports []PortMap, address string) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found with label: %s", label)
	}
	podName := pods.Items[0].Name
	return portForwardResource(clientset, cfg, contextName, namespace, "pod/"+podName, ports, address)
}

func maintainPortForward(cfg *rest.Config, contextName, namespace, podName string, ports []PortMap, address string) {
	portArgs := make([]string, len(ports))
	for i, p := range ports {
		portArgs[i] = fmt.Sprintf("%s:%s", p.Source, p.Target)
	}
	for {
		if err := startPortForward(cfg, contextName, namespace, podName, address, portArgs); err != nil {
			logrus.Errorf("port-forward failed for %s: %v", podName, err)
		}
		time.Sleep(2 * time.Second)
	}
}

func startPortForward(cfg *rest.Config, contextName, namespace, podName, address string, ports []string) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimPrefix(cfg.Host, "https://")
	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return err
	}
	url := &url.URL{Scheme: "https", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})
	pf, err := portforward.NewOnAddresses(dialer, []string{address}, ports, stopCh, readyCh, io.Discard, io.Discard)
	if err != nil {
		return err
	}

	go func() {
		<-readyCh
		logrus.Info(aurora.Green(aurora.Sprintf("Started port-forward for pod %s on %v", aurora.Yellow(aurora.Bold(podName)), aurora.Cyan(aurora.Bold(ports)))))
		equiv := fmt.Sprintf("kubectl --context %s -n %s port-forward pod/%s %s --address %s", contextName, namespace, podName, strings.Join(ports, " "), address)
		logrus.Info(aurora.Yellow(aurora.Sprintf("Equivalent kubectl command: %s", aurora.Cyan(equiv))))
	}()

	err = pf.ForwardPorts()
	close(stopCh)
	return err
}
