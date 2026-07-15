package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/courtyard-nft/go-shippo/models"
)

// CreateManifest creates a new manifest object.
func (c *Client) CreateManifest(ctx context.Context, input *models.ManifestInput) (*models.Manifest, error) {
	if input == nil {
		return nil, errors.New("nil input")
	}

	output := &models.Manifest{}
	err := c.do(ctx, http.MethodPost, "/manifests/", input, output)
	return output, err
}

// RetrieveManifest retrieves an existing manifest by object id.
func (c *Client) RetrieveManifest(ctx context.Context, objectID string) (*models.Manifest, error) {
	if objectID == "" {
		return nil, errors.New("empty object ID")
	}

	output := &models.Manifest{}
	err := c.do(ctx, http.MethodGet, "/manifests/"+objectID, nil, output)
	return output, err
}

// ListAllManifests lists all manifest objects.
func (c *Client) ListAllManifests(ctx context.Context) ([]*models.Manifest, error) {
	list := []*models.Manifest{}
	err := c.doList(ctx, http.MethodGet, "/manifests/", nil, func(v json.RawMessage) error {
		item := &models.Manifest{}
		if err := json.Unmarshal(v, item); err != nil {
			return err
		}

		list = append(list, item)
		return nil
	})
	return list, err
}
