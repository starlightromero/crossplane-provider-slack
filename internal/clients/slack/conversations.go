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
