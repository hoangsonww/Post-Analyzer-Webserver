package adapt

import (
	"testing"
	"time"

	"Post_Analyzer_Webserver/internal/models"
	postgen "Post_Analyzer_Webserver/kitex_gen/post"
)

func TestModelToThriftPost_RoundTrip(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := created.Add(time.Hour)
	m := models.Post{ID: 7, UserID: 3, Title: "Hi", Body: "There", CreatedAt: created, UpdatedAt: updated}

	thrift := ModelToThriftPost(m)
	if thrift.Id != 7 || thrift.UserId != 3 || thrift.Title != "Hi" || thrift.Body != "There" {
		t.Fatalf("unexpected thrift post: %+v", thrift)
	}

	back := ThriftPostToModel(thrift)
	if back.ID != m.ID || back.UserID != m.UserID || back.Title != m.Title || back.Body != m.Body {
		t.Fatalf("round trip mismatch: got %+v, want %+v", back, m)
	}
	if !back.CreatedAt.Equal(created) || !back.UpdatedAt.Equal(updated) {
		t.Fatalf("timestamps didn't round-trip: got created=%v updated=%v", back.CreatedAt, back.UpdatedAt)
	}
}

func TestThriftPostToModel_Nil(t *testing.T) {
	got := ThriftPostToModel(nil)
	if got != (models.Post{}) {
		t.Errorf("expected zero value for nil input, got %+v", got)
	}
}

func TestFilterConversion_RoundTrip(t *testing.T) {
	uid := 5
	m := &models.PostFilter{UserID: &uid, Search: "hello", SortBy: "title", SortOrder: "asc"}

	thrift := ModelFilterToThrift(m)
	if thrift.UserId == nil || *thrift.UserId != 5 {
		t.Fatalf("expected UserId=5, got %v", thrift.UserId)
	}
	if thrift.Search == nil || *thrift.Search != "hello" {
		t.Fatalf("expected Search=hello, got %v", thrift.Search)
	}

	back := ThriftFilterToModel(thrift)
	if back.UserID == nil || *back.UserID != 5 {
		t.Fatalf("expected round-tripped UserID=5, got %v", back.UserID)
	}
	if back.Search != "hello" || back.SortBy != "title" || back.SortOrder != "asc" {
		t.Fatalf("unexpected round-tripped filter: %+v", back)
	}
}

func TestModelFilterToThrift_NilAndEmpty(t *testing.T) {
	if ModelFilterToThrift(nil) != nil {
		t.Error("expected nil filter to convert to nil")
	}
	// An empty (non-nil) filter should produce a thrift filter with all
	// optional fields left unset, not zero-valued-but-present.
	thrift := ModelFilterToThrift(&models.PostFilter{})
	if thrift.UserId != nil || thrift.Search != nil || thrift.SortBy != nil || thrift.SortOrder != nil {
		t.Errorf("expected all-empty filter to leave optional fields nil, got %+v", thrift)
	}
}

func TestThriftPaginationToModel_DefaultsAndOffset(t *testing.T) {
	// Page/PageSize <= 0 should be normalized rather than producing a
	// negative offset.
	got := ThriftPaginationToModel(&postgen.Pagination{Page: 0, PageSize: 0})
	if got.Page != 1 || got.PageSize != 20 || got.Offset != 0 {
		t.Errorf("expected normalized defaults, got %+v", got)
	}

	got2 := ThriftPaginationToModel(&postgen.Pagination{Page: 3, PageSize: 10})
	if got2.Offset != 20 {
		t.Errorf("expected offset=20 for page=3,pageSize=10, got %d", got2.Offset)
	}
}

func TestModelPaginationToThrift_Nil(t *testing.T) {
	got := ModelPaginationToThrift(nil)
	if got == nil {
		t.Fatal("expected a non-nil zero-value PaginationMeta for nil input")
	}
	if got.Page != 0 || got.TotalItems != 0 {
		t.Errorf("expected zero-value fields, got %+v", got)
	}
}

func TestOKAndErr(t *testing.T) {
	ok := OK()
	if ok.StatusCode != 0 || ok.StatusMessage != "OK" {
		t.Errorf("unexpected OK(): %+v", ok)
	}

	e := Err(errFixture{})
	if e.StatusCode == 0 {
		t.Errorf("expected non-zero status code for Err(), got %+v", e)
	}
	if e.StatusMessage != "boom" {
		t.Errorf("expected status message 'boom', got %q", e.StatusMessage)
	}
}

type errFixture struct{}

func (errFixture) Error() string { return "boom" }
