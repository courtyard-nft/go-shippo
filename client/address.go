package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/courtyard-nft/go-shippo/models"
)

// CreateAddress creates a new address object.
func (c *Client) CreateAddress(ctx context.Context, input *models.AddressInput) (*models.Address, error) {
	if input == nil {
		return nil, errors.New("nil input")
	}

	output := &models.Address{}
	err := c.do(ctx, http.MethodPost, "/addresses/", input, output)
	return output, err
}

// RetrieveAddress retrieves an existing address by object id.
func (c *Client) RetrieveAddress(ctx context.Context, objectID string) (*models.Address, error) {
	if objectID == "" {
		return nil, errors.New("empty object ID")
	}

	output := &models.Address{}
	err := c.do(ctx, http.MethodGet, "/addresses/"+objectID, nil, output)
	return output, err
}

// ListAllAddresses lists all addresses.
func (c *Client) ListAllAddresses(ctx context.Context) ([]*models.Address, error) {
	list := []*models.Address{}
	err := c.doList(ctx, http.MethodGet, "/addresses/", nil, func(v json.RawMessage) error {
		item := &models.Address{}
		if err := json.Unmarshal(v, item); err != nil {
			return err
		}

		list = append(list, item)
		return nil
	})
	return list, err
}
