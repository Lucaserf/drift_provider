package ctrldrift

import (
	"os"

	"golang.org/x/crypto/ssh"
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

func (c *external) connect_ssh(address string, ssh_key_path string) (*ssh.Client, error) {

	key, err := os.ReadFile(ssh_key_path)
	if err != nil {
		c.logger.Debug("Error in reading private key")
		c.logger.Debug(err.Error())
		return nil, err
	}
	//parse private key with passphrase
	signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte("/Serf1l1pp1/"))
	if err != nil {
		c.logger.Debug("Error in parsing private key")
		c.logger.Debug(err.Error())
		return nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User: "lucaserf",
		Auth: []ssh.AuthMethod{
			//sshkey
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		c.logger.Debug("Error in creating client")
		c.logger.Debug(err.Error())
		return nil, err
	}

	return client, nil
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
