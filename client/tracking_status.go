package client

import (
	"context"
	"errors"
	"net/http"

	"github.com/courtyard-nft/go-shippo/models"
)

// GetTrackingUpdate requests the tracking status of a shipment.
func (c *Client) GetTrackingUpdate(ctx context.Context, carrier, trackingNumber string) (*models.TrackingStatus, error) {
	if carrier == "" {
		return nil, errors.New("empty carrier")
	}
	if trackingNumber == "" {
		return nil, errors.New("empty tracking number")
	}

	output := &models.TrackingStatus{}
	err := c.do(ctx, http.MethodGet, "/tracks/"+carrier+"/"+trackingNumber, nil, output)
	return output, err
}

// RegisterTrackingWebhook registers a tracking webhook.
// TODO: documentation on this API endpoint is not clear.
// https://goshippo.com/docs/reference#tracks-create
func (c *Client) RegisterTrackingWebhook(ctx context.Context, carrier, trackingNumber, metadata string) (*models.TrackingStatus, error) {
	if carrier == "" {
		return nil, errors.New("empty carrier")
	}
	if trackingNumber == "" {
		return nil, errors.New("empty tracking number")
	}

	output := &models.TrackingStatus{}
	err := c.do(ctx, http.MethodPost, "/tracks/", &models.TrackingStatusInput{
		Carrier:        carrier,
		TrackingNumber: trackingNumber,
		Metadata:       metadata,
	}, output)
	return output, err
}
