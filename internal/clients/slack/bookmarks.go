package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AddBookmark adds a bookmark to a Slack channel.
func (c *Client) AddBookmark(ctx context.Context, channelID string, params BookmarkParams) (*Bookmark, error) {
	vals := url.Values{
		"channel_id": {channelID},
		"title":      {params.Title},
		"type":       {params.Type},
		"link":       {params.Link},
	}

	raw, err := c.Do(ctx, "bookmarks.add", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Bookmark Bookmark `json:"bookmark"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing bookmarks.add response: %w", err)
	}

	return &resp.Bookmark, nil
}

// ListBookmarks lists bookmarks in a Slack channel.
func (c *Client) ListBookmarks(ctx context.Context, channelID string) ([]Bookmark, error) {
	vals := url.Values{
		"channel_id": {channelID},
	}

	raw, err := c.Do(ctx, "bookmarks.list", vals)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Bookmarks []Bookmark `json:"bookmarks"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing bookmarks.list response: %w", err)
	}

	return resp.Bookmarks, nil
}

// EditBookmark edits a bookmark in a Slack channel.
func (c *Client) EditBookmark(ctx context.Context, channelID, bookmarkID string, params BookmarkParams) error {
	vals := url.Values{
		"channel_id":  {channelID},
		"bookmark_id": {bookmarkID},
	}
	if params.Title != "" {
		vals.Set("title", params.Title)
	}
	if params.Link != "" {
		vals.Set("link", params.Link)
	}

	_, err := c.Do(ctx, "bookmarks.edit", vals)
	return err
}

// RemoveBookmark removes a bookmark from a Slack channel.
func (c *Client) RemoveBookmark(ctx context.Context, channelID, bookmarkID string) error {
	vals := url.Values{
		"channel_id":  {channelID},
		"bookmark_id": {bookmarkID},
	}

	_, err := c.Do(ctx, "bookmarks.remove", vals)
	return err
}
