package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/courtyard-nft/go-shippo/models"
)

// CreateRefund creates a new refund object.
func (c *Client) CreateRefund(ctx context.Context, input *models.RefundInput) (*models.Refund, error) {
	if input == nil {
		return nil, errors.New("nil input")
	}

	output := &models.Refund{}
	err := c.do(ctx, http.MethodPost, "/refunds/", input, output)
	return output, err
}

// RetrieveRefund retrieves an existing refund by object id.
func (c *Client) RetrieveRefund(ctx context.Context, objectID string) (*models.Refund, error) {
	if objectID == "" {
		return nil, errors.New("empty object ID")
	}

	output := &models.Refund{}
	err := c.do(ctx, http.MethodGet, "/refunds/"+objectID, nil, output)
	return output, err
}

// ListAllRefunds list all refund objects.
func (c *Client) ListAllRefunds(ctx context.Context) ([]*models.Refund, error) {
	list := []*models.Refund{}
	err := c.doList(ctx, http.MethodGet, "/refunds/", nil, func(v json.RawMessage) error {
		item := &models.Refund{}
		if err := json.Unmarshal(v, item); err != nil {
			return err
		}

		list = append(list, item)
		return nil
	})
	return list, err
}
