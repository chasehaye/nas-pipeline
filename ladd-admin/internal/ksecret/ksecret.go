package ksecret

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	cs        *kubernetes.Clientset
	namespace string
	name      string
}

func New(namespace, name string) (*Client, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build clientset: %w", err)
	}
	return &Client{cs: cs, namespace: namespace, name: name}, nil
}

func (c *Client) Replace(ctx context.Context, filename string, content []byte) error {
	secrets := c.cs.CoreV1().Secrets(c.namespace)

	sec, err := secrets.Get(ctx, c.name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get secret %s/%s: %w", c.namespace, c.name, err)
	}

	sec.Data = map[string][]byte{filename: content}

	if _, err := secrets.Update(ctx, sec, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update secret %s/%s: %w", c.namespace, c.name, err)
	}
	return nil
}
