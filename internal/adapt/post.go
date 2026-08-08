// Package adapt converts between the internal domain models
// (internal/models) and the Kitex/Thrift wire types generated into
// kitex_gen/. It is the single translation layer used by both the
// postsvc RPC server and the gateway's RPC client, so the two never
// drift out of sync independently.
package adapt

import (
	"time"

	"Post_Analyzer_Webserver/internal/models"
	basegen "Post_Analyzer_Webserver/kitex_gen/base"
	postgen "Post_Analyzer_Webserver/kitex_gen/post"
)

func ModelToThriftPost(p models.Post) *postgen.Post {
	return &postgen.Post{
		Id:        int64(p.ID),
		UserId:    int64(p.UserID),
		Title:     p.Title,
		Body:      p.Body,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}
}

func ModelsToThriftPosts(posts []models.Post) []*postgen.Post {
	out := make([]*postgen.Post, len(posts))
	for i, p := range posts {
		out[i] = ModelToThriftPost(p)
	}
	return out
}

func ThriftPostToModel(p *postgen.Post) models.Post {
	if p == nil {
		return models.Post{}
	}
	created, _ := time.Parse(time.RFC3339, p.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, p.UpdatedAt)
	return models.Post{
		ID:        int(p.Id),
		UserID:    int(p.UserId),
		Title:     p.Title,
		Body:      p.Body,
		CreatedAt: created,
		UpdatedAt: updated,
	}
}

func ThriftFilterToModel(f *postgen.PostFilter) *models.PostFilter {
	if f == nil {
		return nil
	}
	out := &models.PostFilter{}
	if f.UserId != nil {
		uid := int(*f.UserId)
		out.UserID = &uid
	}
	if f.Search != nil {
		out.Search = *f.Search
	}
	if f.SortBy != nil {
		out.SortBy = *f.SortBy
	}
	if f.SortOrder != nil {
		out.SortOrder = *f.SortOrder
	}
	return out
}

func ModelFilterToThrift(f *models.PostFilter) *postgen.PostFilter {
	if f == nil {
		return nil
	}
	out := &postgen.PostFilter{}
	if f.UserID != nil {
		uid := int64(*f.UserID)
		out.UserId = &uid
	}
	if f.Search != "" {
		out.Search = &f.Search
	}
	if f.SortBy != "" {
		out.SortBy = &f.SortBy
	}
	if f.SortOrder != "" {
		out.SortOrder = &f.SortOrder
	}
	return out
}

func ThriftPaginationToModel(p *postgen.Pagination) *models.PaginationParams {
	if p == nil {
		return nil
	}
	page := int(p.Page)
	size := int(p.PageSize)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	return &models.PaginationParams{
		Page:     page,
		PageSize: size,
		Offset:   (page - 1) * size,
	}
}

func ModelPaginationToThrift(p *models.PaginationMeta) *postgen.PaginationMeta {
	if p == nil {
		return &postgen.PaginationMeta{}
	}
	return &postgen.PaginationMeta{
		Page:       int32(p.Page),
		PageSize:   int32(p.PageSize),
		TotalItems: int32(p.TotalItems),
		TotalPages: int32(p.TotalPages),
		HasNext:    p.HasNext,
		HasPrev:    p.HasPrev,
	}
}

func OK() *basegen.BaseResp {
	return &basegen.BaseResp{StatusCode: 0, StatusMessage: "OK"}
}

func Err(err error) *basegen.BaseResp {
	return &basegen.BaseResp{StatusCode: 1, StatusMessage: err.Error()}
}
