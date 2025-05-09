package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type serviceIpManager struct {
	serviceIPs    *sync.Map
	getServiceIPs func() ([]string, error)
}

func newServiceIpManager() *serviceIpManager {
	newManager := &serviceIpManager{
		serviceIPs:    &sync.Map{},
		getServiceIPs: getServiceIPs,
	}
	go newManager.run()
	return newManager
}

func (s *serviceIpManager) run() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			currentServiceIPs, err := s.getServiceIPs()
			if err != nil {
				slog.Error("Failed to get service IPs:", "Err", err.Error())
				continue
			}
			slog.Debug(
				"Discovered service IPs",
				"ServiceIpCount", len(currentServiceIPs),
				"ServiceIPs", currentServiceIPs,
			)
			for _, ip := range currentServiceIPs {
				s.serviceIPs.Store(ip, struct{}{})
			}
			s.serviceIPs.Range(func(key, value interface{}) bool {
				for _, ip := range currentServiceIPs {
					if key.(string) == ip {
						return true
					}
				}
				s.serviceIPs.Delete(key)
				return true
			})
		}
	}
}

func (s *serviceIpManager) isServiceIP(ip string) bool {
	_, ok := s.serviceIPs.Load(ip)
	return ok
}

func getServiceIPs() ([]string, error) {
	// Load config from inside the cluster or from kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("Failed to load kubeconfig: %v", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Kubernetes client: %v", err)
	}

	// Get all services in all namespaces
	services, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to list services: %v", err)
	}

	// Extract service ClusterIPs
	var serviceIPs []string
	for _, svc := range services.Items {
		if svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != "None" {
			serviceIPs = append(serviceIPs, svc.Spec.ClusterIP)
		}
	}

	return serviceIPs, nil
}
