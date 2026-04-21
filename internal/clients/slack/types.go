package slack

// Conversation represents a Slack channel.
type Conversation struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsArchived bool   `json:"is_archived"`
	Topic      Topic  `json:"topic"`
	Purpose    Topic  `json:"purpose"`
	NumMembers int    `json:"num_members"`
	Created    int64  `json:"created"`
}

// Topic represents a channel topic or purpose.
type Topic struct {
	Value string `json:"value"`
}

// Bookmark represents a Slack channel bookmark.
type Bookmark struct {
	ID          string `json:"id"`
	ChannelID   string `json:"channel_id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Link        string `json:"link"`
	DateCreated int64  `json:"date_created"`
}

// BookmarkParams holds parameters for creating or editing a bookmark.
type BookmarkParams struct {
	Title string
	Type  string
	Link  string
}

// Pin represents a pinned item in a Slack channel.
type Pin struct {
	Channel string  `json:"channel"`
	Message Message `json:"message"`
	Created int64   `json:"created"`
}

// Message represents a Slack message (subset of fields).
type Message struct {
	Ts string `json:"ts"`
}

// UserGroup represents a Slack user group.
type UserGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Handle      string `json:"handle"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"is_usergroup"`
	CreatedBy   string `json:"created_by"`
	DateCreate  int64  `json:"date_create"`
}

// UserGroupParams holds parameters for creating or updating a user group.
type UserGroupParams struct {
	Name        string
	Handle      string
	Description string
}

// User represents a Slack user (subset of fields).
type User struct {
	ID      string      `json:"id"`
	Profile UserProfile `json:"profile"`
}

// UserProfile represents a Slack user's profile.
type UserProfile struct {
	Email string `json:"email"`
}
