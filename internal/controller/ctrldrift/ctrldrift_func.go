package ctrldrift

// import (
// 	"context"
// 	"fmt"

// 	"strconv"

// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	"k8s.io/client-go/dynamic"
// 	"k8s.io/client-go/rest"
// )

// func (c *external) connect_kube(address string)(*dynamic.DynamicClient, error) {
// 	config := &rest.Config{
// 		Host: address,
// 	}
// 	client, err := dynamic.NewForConfig(config)
// 	if err != nil {
// 		c.logger.Debug("error in creating new client", "error", err.Error())
// 		return nil, err
// 	}
// 	return client, nil
// }

func (c *external) get_new_data(address string)
