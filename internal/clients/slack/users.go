package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

// LookupUserByEmail looks up a Slack user by email address.
func (c *Client) LookupUserByEmail(ctx context.Context, email string) (*User, error) {
	vals := url.Values{}
	vals.Set("email", email)

	raw, err := c.Do(ctx, "users.lookupByEmail", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}
