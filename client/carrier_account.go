package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/courtyard-nft/go-shippo/models"
)

// CreateCarrierAccount creates a new carrier account object.
func (c *Client) CreateCarrierAccount(ctx context.Context, input *models.CarrierAccountInput) (*models.CarrierAccount, error) {
	if input == nil {
		return nil, errors.New("nil input")
	}

	output := &models.CarrierAccount{}
	err := c.do(ctx, http.MethodPost, "/carrier_accounts/", input, output)
	return output, err
}

// RetrieveCarrierAccount retrieves an existing carrier account by object id.
func (c *Client) RetrieveCarrierAccount(ctx context.Context, objectID string) (*models.CarrierAccount, error) {
	if objectID == "" {
		return nil, errors.New("empty object ID")
	}

	output := &models.CarrierAccount{}
	err := c.do(ctx, http.MethodGet, "/carrier_accounts/"+objectID, nil, output)
	return output, err
}

// ListAllCarrierAccounts lists all carrier accounts.
func (c *Client) ListAllCarrierAccounts(ctx context.Context) ([]*models.CarrierAccount, error) {
	list := []*models.CarrierAccount{}
	err := c.doList(ctx, http.MethodGet, "/carrier_accounts/", nil, func(v json.RawMessage) error {
		item := &models.CarrierAccount{}
		if err := json.Unmarshal(v, item); err != nil {
			return err
		}

		list = append(list, item)
		return nil
	})
	return list, err
}

// UpdateCarrierAccount updates an existing carrier account.
// AccountID and Carrier cannot be updated because they form the unique identifier together.
func (c *Client) UpdateCarrierAccount(ctx context.Context, objectID string, input *models.CarrierAccountInput) (*models.CarrierAccount, error) {
	if objectID == "" {
		return nil, errors.New("empty object ID")
	}
	if input == nil {
		return nil, errors.New("nil input")
	}

	output := &models.CarrierAccount{}
	err := c.do(ctx, http.MethodPut, "/carrier_accounts/"+objectID, input, output)
	return output, err
}
