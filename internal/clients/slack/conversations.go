package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// CreateConversation creates a new Slack channel with the given name and privacy setting.
// It returns the created Conversation with the channel ID populated.
func (c *Client) CreateConversation(ctx context.Context, name string, isPrivate bool) (*Conversation, error) {
	params := url.Values{}
	params.Set("name", name)
	if isPrivate {
		params.Set("is_private", "true")
	} else {
		params.Set("is_private", "false")
	}

	raw, err := c.Do(ctx, "conversations.create", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Channel Conversation `json:"channel"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing conversations.create response: %w", err)
	}

	return &resp.Channel, nil
}

// GetConversationInfo retrieves information about a Slack channel by its ID.
func (c *Client) GetConversationInfo(ctx context.Context, channelID string) (*Conversation, error) {
	params := url.Values{}
	params.Set("channel", channelID)

	raw, err := c.Do(ctx, "conversations.info", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Channel Conversation `json:"channel"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing conversations.info response: %w", err)
	}

	return &resp.Channel, nil
}

// RenameConversation renames a Slack channel.
func (c *Client) RenameConversation(ctx context.Context, channelID, name string) error {
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("name", name)

	_, err := c.Do(ctx, "conversations.rename", params)
	return err
}

// SetConversationTopic sets the topic for a Slack channel.
func (c *Client) SetConversationTopic(ctx context.Context, channelID, topic string) error {
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("topic", topic)

	_, err := c.Do(ctx, "conversations.setTopic", params)
	return err
}

// SetConversationPurpose sets the purpose for a Slack channel.
func (c *Client) SetConversationPurpose(ctx context.Context, channelID, purpose string) error {
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("purpose", purpose)

	_, err := c.Do(ctx, "conversations.setPurpose", params)
	return err
}

// ArchiveConversation archives a Slack channel.
func (c *Client) ArchiveConversation(ctx context.Context, channelID string) error {
	params := url.Values{}
	params.Set("channel", channelID)

	_, err := c.Do(ctx, "conversations.archive", params)
	return err
}

// FindConversationByName searches for a channel by name using conversations.list.
// Returns the Conversation if found, or nil if not found.
func (c *Client) FindConversationByName(ctx context.Context, name string) (*Conversation, error) {
	var cursor string
	for {
		params := url.Values{}
		params.Set("limit", "200")
		params.Set("exclude_archived", "false")
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		raw, err := c.Do(ctx, "conversations.list", params)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Channels []Conversation `json:"channels"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("parsing conversations.list response: %w", err)
		}

		for i := range resp.Channels {
			if resp.Channels[i].Name == name {
				return &resp.Channels[i], nil
			}
		}

		cursor = resp.Metadata.NextCursor
		if cursor == "" {
			break
		}
	}

	return nil, nil
}

// JoinConversation joins the bot to a Slack channel.
func (c *Client) JoinConversation(ctx context.Context, channelID string) error {
	params := url.Values{}
	params.Set("channel", channelID)

	_, err := c.Do(ctx, "conversations.join", params)
	return err
}

// InviteToConversation invites a user to a Slack channel.
func (c *Client) InviteToConversation(ctx context.Context, channelID, userID string) error {
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("users", userID)

	_, err := c.Do(ctx, "conversations.invite", params)
	return err
}

// KickFromConversation removes a user from a Slack channel.
func (c *Client) KickFromConversation(ctx context.Context, channelID, userID string) error {
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("user", userID)

	_, err := c.Do(ctx, "conversations.kick", params)
	return err
}

// GetConversationMembers returns the list of user IDs in a channel.
func (c *Client) GetConversationMembers(ctx context.Context, channelID string) ([]string, error) {
	var members []string
	var cursor string
	for {
		params := url.Values{}
		params.Set("channel", channelID)
		params.Set("limit", "200")
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		raw, err := c.Do(ctx, "conversations.members", params)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Members  []string `json:"members"`
			Metadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("parsing conversations.members response: %w", err)
		}

		members = append(members, resp.Members...)
		cursor = resp.Metadata.NextCursor
		if cursor == "" {
			break
		}
	}
	return members, nil
}
