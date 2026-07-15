package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/courtyard-nft/go-shippo/models"
)

// GetShippingRates gets rates for a shipping object.
func (c *Client) GetShippingRates(ctx context.Context, shipmentObjectID, currencyCode string) ([]*models.Rate, error) {
	if shipmentObjectID == "" {
		return nil, errors.New("empty shipment object ID")
	}
	if currencyCode == "" {
		return nil, errors.New("empty currency code")
	}

	list := []*models.Rate{}
	err := c.doList(ctx, http.MethodGet, "/shipments/"+shipmentObjectID+"/rates/"+currencyCode, nil, func(v json.RawMessage) error {
		item := &models.Rate{}
		if err := json.Unmarshal(v, item); err != nil {
			return err
		}

		list = append(list, item)
		return nil
	})
	return list, err
}

// RetrieveRate retrieves an existing rate by object id.
func (c *Client) RetrieveRate(ctx context.Context, objectID string) (*models.Rate, error) {
	if objectID == "" {
		return nil, errors.New("empty object ID")
	}

	output := &models.Rate{}
	err := c.do(ctx, http.MethodGet, "/rates/"+objectID, nil, output)
	return output, err
}
