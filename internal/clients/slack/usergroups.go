package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

// CreateUserGroup creates a new Slack user group.
func (c *Client) CreateUserGroup(ctx context.Context, params UserGroupParams) (*UserGroup, error) {
	vals := url.Values{}
	vals.Set("name", params.Name)
	vals.Set("handle", params.Handle)
	if params.Description != "" {
		vals.Set("description", params.Description)
	}

	raw, err := c.Do(ctx, "usergroups.create", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		UserGroup UserGroup `json:"usergroup"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp.UserGroup, nil
}

// ListUserGroups lists all user groups in the workspace.
func (c *Client) ListUserGroups(ctx context.Context) ([]UserGroup, error) {
	vals := url.Values{}
	vals.Set("include_disabled", "true")

	raw, err := c.Do(ctx, "usergroups.list", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		UserGroups []UserGroup `json:"usergroups"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp.UserGroups, nil
}

// UpdateUserGroup updates a Slack user group.
func (c *Client) UpdateUserGroup(ctx context.Context, groupID string, params UserGroupParams) error {
	vals := url.Values{}
	vals.Set("usergroup", groupID)
	vals.Set("name", params.Name)
	vals.Set("handle", params.Handle)
	if params.Description != "" {
		vals.Set("description", params.Description)
	}

	_, err := c.Do(ctx, "usergroups.update", vals)
	return err
}

// DisableUserGroup disables a Slack user group.
func (c *Client) DisableUserGroup(ctx context.Context, groupID string) error {
	vals := url.Values{}
	vals.Set("usergroup", groupID)

	_, err := c.Do(ctx, "usergroups.disable", vals)
	return err
}

// ListUserGroupMembers lists members of a Slack user group.
func (c *Client) ListUserGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	vals := url.Values{}
	vals.Set("usergroup", groupID)

	raw, err := c.Do(ctx, "usergroups.users.list", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Users []string `json:"users"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp.Users, nil
}

// UpdateUserGroupMembers updates the members of a Slack user group.
func (c *Client) UpdateUserGroupMembers(ctx context.Context, groupID string, userIDs []string) error {
	vals := url.Values{}
	vals.Set("usergroup", groupID)

	// Join user IDs with commas
	users := ""
	for i, id := range userIDs {
		if i > 0 {
			users += ","
		}
		users += id
	}
	vals.Set("users", users)

	_, err := c.Do(ctx, "usergroups.users.update", vals)
	return err
}
