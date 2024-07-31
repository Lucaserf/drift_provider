package ctrldrift

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (c *external) connect_dynamic(address string) (*dynamic.DynamicClient, error) {
	//for real implementation download kubeconfig file from gitlab given address

	config := &rest.Config{
		Host: address,
	}

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		c.logger.Debug("Error in creating clientset")
		c.logger.Debug(err.Error())
		return nil, err
	}
	return clientset, nil

}

func (c *external) connect_kube_client() (*kubernetes.Clientset, error) {
	//for real implementation download kubeconfig file from gitlab given address

	// config := &rest.Config{
	// 	Host: address,
	// }
	config, err := rest.InClusterConfig()
	if err != nil {
		c.logger.Debug("Error in getting in cluster config")
		c.logger.Debug(err.Error())
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		c.logger.Debug("Error in creating clientset")
		c.logger.Debug(err.Error())
		return nil, err
	}
	return clientset, nil

}
