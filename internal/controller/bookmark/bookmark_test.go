/*
Copyright 2024 Avodah Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bookmark

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/avodah-inc/crossplane-provider-slack/internal/clients/slack"
)

func TestObserve_ExistingBookmark(t *testing.T) {
	channelID := "C12345678"
	bookmarkID := "Bk00000001"

	mock := &mockClientAPI{
		listBookmarksFn: func(_ context.Context, _ string) ([]slack.Bookmark, error) {
			return []slack.Bookmark{
				{ID: bookmarkID, ChannelID: channelID, Title: "My Link", Link: "https://example.com", Type: "link", DateCreated: 1700000000},
			}, nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Link", "https://example.com")
	meta.SetExternalName(cr, bookmarkID)

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true")
	}
	if cr.Status.AtProvider.ID != bookmarkID {
		t.Fatalf("expected AtProvider.ID=%s, got %q", bookmarkID, cr.Status.AtProvider.ID)
	}
	if cr.Status.AtProvider.ChannelID != channelID {
		t.Fatalf("expected AtProvider.ChannelID=%s, got %q", channelID, cr.Status.AtProvider.ChannelID)
	}
}

func TestObserve_BookmarkNotFound(t *testing.T) {
	channelID := "C12345678"

	mock := &mockClientAPI{
		listBookmarksFn: func(_ context.Context, _ string) ([]slack.Bookmark, error) {
			return []slack.Bookmark{}, nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "Missing", "https://example.com")
	meta.SetExternalName(cr, "BkNOTFOUND")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_NoExternalName(t *testing.T) {
	mock := &mockClientAPI{}
	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR("C12345678", "New Bookmark", "https://example.com")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false when no external-name")
	}
}

func TestObserve_DriftDetection(t *testing.T) {
	channelID := "C12345678"
	bookmarkID := "Bk00000001"

	mock := &mockClientAPI{
		listBookmarksFn: func(_ context.Context, _ string) ([]slack.Bookmark, error) {
			return []slack.Bookmark{
				{ID: bookmarkID, ChannelID: channelID, Title: "Old Title", Link: "https://old.com", Type: "link"},
			}, nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "New Title", "https://new.com")
	meta.SetExternalName(cr, bookmarkID)

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when title and link differ")
	}
}

func TestCreate_Success(t *testing.T) {
	channelID := "C12345678"
	newBookmarkID := "BkNEW00001"

	mock := &mockClientAPI{
		addBookmarkFn: func(_ context.Context, _ string, params slack.BookmarkParams) (*slack.Bookmark, error) {
			return &slack.Bookmark{
				ID:        newBookmarkID,
				ChannelID: channelID,
				Title:     params.Title,
				Type:      params.Type,
				Link:      params.Link,
			}, nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Bookmark", "https://example.com")

	_, err := ext.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got := meta.GetExternalName(cr)
	if got != newBookmarkID {
		t.Fatalf("external-name = %q, want %q", got, newBookmarkID)
	}
}

func TestUpdate_ChangedTitle(t *testing.T) {
	channelID := "C12345678"
	bookmarkID := "Bk00000001"
	var editCalled bool

	mock := &mockClientAPI{
		editBookmarkFn: func(_ context.Context, _, _ string, params slack.BookmarkParams) error {
			editCalled = true
			if params.Title != "New Title" {
				t.Fatalf("edit title = %q, want New Title", params.Title)
			}
			return nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "New Title", "https://example.com")
	meta.SetExternalName(cr, bookmarkID)

	_, err := ext.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !editCalled {
		t.Fatal("expected EditBookmark to be called")
	}
}

func TestDelete_Success(t *testing.T) {
	channelID := "C12345678"
	bookmarkID := "Bk00000001"
	var removeCalled bool

	mock := &mockClientAPI{
		removeBookmarkFn: func(_ context.Context, chID, bmID string) error {
			removeCalled = true
			if chID != channelID {
				t.Fatalf("remove channelID = %q, want %q", chID, channelID)
			}
			if bmID != bookmarkID {
				t.Fatalf("remove bookmarkID = %q, want %q", bmID, bookmarkID)
			}
			return nil
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Bookmark", "https://example.com")
	meta.SetExternalName(cr, bookmarkID)

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !removeCalled {
		t.Fatal("expected RemoveBookmark to be called")
	}
}

func TestObserve_ChannelNotFound(t *testing.T) {
	channelID := "C12345678"

	mock := &mockClientAPI{
		listBookmarksFn: func(_ context.Context, _ string) ([]slack.Bookmark, error) {
			return nil, &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Bookmark", "https://example.com")
	meta.SetExternalName(cr, "Bk00000001")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false for channel_not_found")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonChannelUnavailable {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonChannelUnavailable, cond.Reason)
	}
}

func TestObserve_IsArchived(t *testing.T) {
	channelID := "C12345678"

	mock := &mockClientAPI{
		listBookmarksFn: func(_ context.Context, _ string) ([]slack.Bookmark, error) {
			return nil, &slack.SlackError{Code: "is_archived", Message: "channel is archived"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Bookmark", "https://example.com")
	meta.SetExternalName(cr, "Bk00000001")

	obs, err := ext.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false for is_archived")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonChannelUnavailable {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonChannelUnavailable, cond.Reason)
	}
}

func TestCreate_ChannelNotFound(t *testing.T) {
	channelID := "C12345678"

	mock := &mockClientAPI{
		addBookmarkFn: func(_ context.Context, _ string, _ slack.BookmarkParams) (*slack.Bookmark, error) {
			return nil, &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Bookmark", "https://example.com")

	_, err := ext.Create(context.Background(), cr)
	if err == nil {
		t.Fatal("expected error from Create with channel_not_found")
	}

	cond := cr.GetCondition(xpv1.TypeSynced)
	if cond.Reason != ReasonChannelUnavailable {
		t.Fatalf("expected Synced condition reason=%s, got %s", ReasonChannelUnavailable, cond.Reason)
	}
}

func TestDelete_ChannelNotFound(t *testing.T) {
	channelID := "C12345678"

	mock := &mockClientAPI{
		removeBookmarkFn: func(_ context.Context, _, _ string) error {
			return &slack.SlackError{Code: "channel_not_found", Message: "channel not found"}
		},
	}

	ext := &external{client: mock, kube: nil}
	cr := newBookmarkCR(channelID, "My Bookmark", "https://example.com")
	meta.SetExternalName(cr, "Bk00000001")

	_, err := ext.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete should not return error for channel_not_found, got: %v", err)
	}
}
