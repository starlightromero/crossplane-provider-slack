package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the default Slack Web API base URL.
	DefaultBaseURL = "https://slack.com/api"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second
)

// ClientAPI defines the Slack Web API operations used by controllers.
type ClientAPI interface {
	// Conversations
	CreateConversation(ctx context.Context, name string, isPrivate bool) (*Conversation, error)
	GetConversationInfo(ctx context.Context, channelID string) (*Conversation, error)
	RenameConversation(ctx context.Context, channelID, name string) error
	SetConversationTopic(ctx context.Context, channelID, topic string) error
	SetConversationPurpose(ctx context.Context, channelID, purpose string) error
	ArchiveConversation(ctx context.Context, channelID string) error

	// Bookmarks
	AddBookmark(ctx context.Context, channelID string, params BookmarkParams) (*Bookmark, error)
	ListBookmarks(ctx context.Context, channelID string) ([]Bookmark, error)
	EditBookmark(ctx context.Context, channelID, bookmarkID string, params BookmarkParams) error
	RemoveBookmark(ctx context.Context, channelID, bookmarkID string) error

	// Pins
	AddPin(ctx context.Context, channelID, messageTS string) error
	ListPins(ctx context.Context, channelID string) ([]Pin, error)
	RemovePin(ctx context.Context, channelID, messageTS string) error

	// User Groups
	CreateUserGroup(ctx context.Context, params UserGroupParams) (*UserGroup, error)
	ListUserGroups(ctx context.Context) ([]UserGroup, error)
	UpdateUserGroup(ctx context.Context, groupID string, params UserGroupParams) error
	DisableUserGroup(ctx context.Context, groupID string) error

	// User Group Members
	ListUserGroupMembers(ctx context.Context, groupID string) ([]string, error)
	UpdateUserGroupMembers(ctx context.Context, groupID string, userIDs []string) error

	// Users
	LookupUserByEmail(ctx context.Context, email string) (*User, error)
}

// Client implements ClientAPI using the Slack Web API over HTTP.
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the Slack API.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates a new Slack API client with the given bot token.
func NewClient(token string, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		token:      token,
		baseURL:    DefaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// apiResponse is the common envelope for Slack API JSON responses.
type apiResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Raw   json.RawMessage `json:"-"`
}

// Do executes a Slack API call. It builds a POST request to baseURL/method
// with the given form parameters, sets the Authorization header, handles
// rate limiting with retries, parses the JSON response, and checks the "ok" field.
func (c *Client) Do(ctx context.Context, method string, params url.Values) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("%s/%s", c.baseURL, method)

	var body io.Reader
	if params != nil {
		body = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", method, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Store encoded params for potential retries (body is consumed on first read).
	var encodedParams string
	if params != nil {
		encodedParams = params.Encode()
	}
	req.GetBody = func() (io.ReadCloser, error) {
		if encodedParams == "" {
			return http.NoBody, nil
		}
		return io.NopCloser(strings.NewReader(encodedParams)), nil
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		// Wrap network errors as retriable for the reconciler.
		if IsNetworkError(err) {
			return nil, &RetriableError{Err: fmt.Errorf("executing request for %s: %w", method, err)}
		}
		return nil, fmt.Errorf("executing request for %s: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body for %s: %w", method, err)
	}

	var envelope struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parsing response for %s: %w", method, err)
	}

	if !envelope.OK {
		return nil, &SlackError{
			Code:    envelope.Error,
			Message: fmt.Sprintf("slack API %s returned error: %s", method, envelope.Error),
		}
	}

	return json.RawMessage(respBody), nil
}
