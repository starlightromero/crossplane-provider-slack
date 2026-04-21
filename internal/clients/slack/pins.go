package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AddPin pins a message in a Slack channel.
func (c *Client) AddPin(ctx context.Context, channelID, messageTS string) error {
	vals := url.Values{
		"channel":   {channelID},
		"timestamp": {messageTS},
	}

	_, err := c.Do(ctx, "pins.add", vals)
	return err
}

// ListPins lists pinned items in a Slack channel.
func (c *Client) ListPins(ctx context.Context, channelID string) ([]Pin, error) {
	vals := url.Values{
		"channel": {channelID},
	}

	raw, err := c.Do(ctx, "pins.list", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			Message Message `json:"message"`
			Created int64   `json:"created"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing pins.list response: %w", err)
	}

	pins := make([]Pin, len(resp.Items))
	for i, item := range resp.Items {
		pins[i] = Pin{
			Channel: channelID,
			Message: item.Message,
			Created: item.Created,
		}
	}

	return pins, nil
}

// RemovePin removes a pin from a Slack channel.
func (c *Client) RemovePin(ctx context.Context, channelID, messageTS string) error {
	vals := url.Values{
		"channel":   {channelID},
		"timestamp": {messageTS},
	}

	_, err := c.Do(ctx, "pins.remove", vals)
	return err
}
