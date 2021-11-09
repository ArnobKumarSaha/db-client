package dbclient

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
)

type Client struct {
	*mongo.Client
}

func (c *Client) Close() {
	err := c.Disconnect(context.TODO())
	if err != nil {
		klog.Errorf("Failed to disconnect client. error: %v", err)
		return
	}
}
