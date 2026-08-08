// Package rpcclient wraps the generated Kitex clients behind small
// Go-native interfaces, so the gateway's HTTP handlers (internal/api,
// internal/handlers, internal/cli) can call postsvc/authsvc exactly like
// they'd call an in-process service — the RPC hop is an implementation
// detail hidden here.
package rpcclient

import (
	"context"
	"fmt"

	"Post_Analyzer_Webserver/internal/adapt"
	"Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/models"
	basegen "Post_Analyzer_Webserver/kitex_gen/base"
	post "Post_Analyzer_Webserver/kitex_gen/post"
	postservice "Post_Analyzer_Webserver/kitex_gen/post/postservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/transport"
)

// PostClient is the gateway-facing view of the post-analysis RPC service.
// Its method set intentionally mirrors service.PostService so callers
// don't need to know whether they're talking to an in-process service or
// a Kitex RPC backend.
type PostClient interface {
	GetAll(ctx context.Context, filter *models.PostFilter, pagination *models.PaginationParams) ([]models.Post, *models.PaginationMeta, error)
	GetByID(ctx context.Context, id int) (*models.Post, error)
	Create(ctx context.Context, req *models.CreatePostRequest) (*models.Post, error)
	Update(ctx context.Context, id int, req *models.UpdatePostRequest) (*models.Post, error)
	Delete(ctx context.Context, id int) error
	BulkCreate(ctx context.Context, req *models.BulkCreateRequest) (*models.BulkCreateResponse, error)
	AnalyzeCharacterFrequency(ctx context.Context) (*models.AnalyticsResult, error)
}

type postRPCClient struct {
	cli postservice.Client
}

// NewPostClient dials the postsvc Kitex server. mux enables connection
// multiplexing (one TCP connection shared across concurrent RPCs) instead
// of Kitex's default one-connection-per-call pool.
func NewPostClient(addr string, mux bool) (PostClient, error) {
	opts := []client.Option{
		client.WithHostPorts(addr),
		client.WithTransportProtocol(transport.TTHeader),
	}
	if mux {
		opts = append(opts, client.WithMuxConnection(1))
	}

	cli, err := postservice.NewClient("postsvc", opts...)
	if err != nil {
		return nil, fmt.Errorf("dial postsvc at %s: %w", addr, err)
	}
	return &postRPCClient{cli: cli}, nil
}

func newBase() *basegen.Base {
	return &basegen.Base{Client: "gateway"}
}

// rpcErr turns a transport-level error or a non-OK BaseResp into a single
// AppError. baseResp is nil-safe: pass nil when err != nil since the
// generated response pointer may itself be nil on transport failure.
func rpcErr(err error, baseResp *basegen.BaseResp) error {
	if err != nil {
		return errors.NewInternalError(fmt.Errorf("postsvc rpc: %w", err))
	}
	if baseResp != nil && baseResp.StatusCode != 0 {
		return errors.NewInternalError(fmt.Errorf("postsvc: %s", baseResp.StatusMessage))
	}
	return nil
}

func (c *postRPCClient) GetAll(ctx context.Context, filter *models.PostFilter, pagination *models.PaginationParams) ([]models.Post, *models.PaginationMeta, error) {
	req := &post.ListPostsRequest{
		Base:   newBase(),
		Filter: adapt.ModelFilterToThrift(filter),
	}
	if pagination != nil {
		req.Pagination = &post.Pagination{Page: int32(pagination.Page), PageSize: int32(pagination.PageSize)}
	}
	resp, err := c.cli.ListPosts(ctx, req)
	if err != nil {
		return nil, nil, rpcErr(err, nil)
	}
	if e := rpcErr(nil, resp.BaseResp); e != nil {
		return nil, nil, e
	}
	posts := make([]models.Post, len(resp.Posts))
	for i, p := range resp.Posts {
		posts[i] = adapt.ThriftPostToModel(p)
	}
	meta := &models.PaginationMeta{
		Page: int(resp.Pagination.Page), PageSize: int(resp.Pagination.PageSize),
		TotalItems: int(resp.Pagination.TotalItems), TotalPages: int(resp.Pagination.TotalPages),
		HasNext: resp.Pagination.HasNext, HasPrev: resp.Pagination.HasPrev,
	}
	return posts, meta, nil
}

func (c *postRPCClient) GetByID(ctx context.Context, id int) (*models.Post, error) {
	resp, err := c.cli.GetPost(ctx, &post.GetPostRequest{Base: newBase(), Id: int64(id)})
	if err != nil {
		return nil, rpcErr(err, nil)
	}
	if e := rpcErr(nil, resp.BaseResp); e != nil {
		return nil, e
	}
	p := adapt.ThriftPostToModel(resp.Post)
	return &p, nil
}

func (c *postRPCClient) Create(ctx context.Context, req *models.CreatePostRequest) (*models.Post, error) {
	resp, err := c.cli.CreatePost(ctx, &post.CreatePostRequest{
		Base: newBase(), UserId: int64(req.UserID), Title: req.Title, Body: req.Body,
	})
	if err != nil {
		return nil, rpcErr(err, nil)
	}
	if e := rpcErr(nil, resp.BaseResp); e != nil {
		return nil, e
	}
	p := adapt.ThriftPostToModel(resp.Post)
	return &p, nil
}

func (c *postRPCClient) Update(ctx context.Context, id int, req *models.UpdatePostRequest) (*models.Post, error) {
	treq := &post.UpdatePostRequest{Base: newBase(), Id: int64(id)}
	if req.Title != "" {
		treq.Title = &req.Title
	}
	if req.Body != "" {
		treq.Body = &req.Body
	}
	resp, err := c.cli.UpdatePost(ctx, treq)
	if err != nil {
		return nil, rpcErr(err, nil)
	}
	if e := rpcErr(nil, resp.BaseResp); e != nil {
		return nil, e
	}
	p := adapt.ThriftPostToModel(resp.Post)
	return &p, nil
}

func (c *postRPCClient) Delete(ctx context.Context, id int) error {
	resp, err := c.cli.DeletePost(ctx, &post.DeletePostRequest{Base: newBase(), Id: int64(id)})
	if err != nil {
		return rpcErr(err, nil)
	}
	return rpcErr(nil, resp.BaseResp)
}

func (c *postRPCClient) BulkCreate(ctx context.Context, req *models.BulkCreateRequest) (*models.BulkCreateResponse, error) {
	treq := &post.BatchCreatePostsRequest{Base: newBase(), Posts: make([]*post.CreatePostRequest, len(req.Posts))}
	for i, p := range req.Posts {
		treq.Posts[i] = &post.CreatePostRequest{Base: newBase(), UserId: int64(p.UserID), Title: p.Title, Body: p.Body}
	}
	resp, err := c.cli.BatchCreatePosts(ctx, treq)
	if err != nil {
		return nil, rpcErr(err, nil)
	}
	if e := rpcErr(nil, resp.BaseResp); e != nil {
		return nil, e
	}
	postIDs := make([]int, len(resp.PostIds))
	for i, id := range resp.PostIds {
		postIDs[i] = int(id)
	}
	return &models.BulkCreateResponse{
		Created: int(resp.Created), Failed: int(resp.Failed), Errors: resp.Errors, PostIDs: postIDs,
	}, nil
}

func (c *postRPCClient) AnalyzeCharacterFrequency(ctx context.Context) (*models.AnalyticsResult, error) {
	resp, err := c.cli.AnalyzePosts(ctx, &post.AnalyzePostsRequest{Base: newBase()})
	if err != nil {
		return nil, rpcErr(err, nil)
	}
	if e := rpcErr(nil, resp.BaseResp); e != nil {
		return nil, e
	}
	topChars := make([]models.CharacterStat, len(resp.TopCharacters))
	for i, cs := range resp.TopCharacters {
		r := []rune(cs.Character)
		var ch rune
		if len(r) > 0 {
			ch = r[0]
		}
		topChars[i] = models.CharacterStat{Character: ch, Count: int(cs.Count), Frequency: cs.Frequency}
	}
	stats := &models.AnalyticsStats{}
	if resp.Statistics != nil {
		stats.AveragePostLength = resp.Statistics.AveragePostLength
		stats.MedianPostLength = int(resp.Statistics.MedianPostLength)
		stats.PostsPerUser = make(map[int]int, len(resp.Statistics.PostsPerUser))
		for uid, n := range resp.Statistics.PostsPerUser {
			stats.PostsPerUser[int(uid)] = int(n)
		}
		stats.TimeDistribution = make(map[string]int, len(resp.Statistics.TimeDistribution))
		for k, v := range resp.Statistics.TimeDistribution {
			stats.TimeDistribution[k] = int(v)
		}
	}
	return &models.AnalyticsResult{
		TotalPosts:      int(resp.TotalPosts),
		TotalCharacters: int(resp.TotalCharacters),
		UniqueChars:     int(resp.UniqueChars),
		TopCharacters:   topChars,
		Statistics:      stats,
	}, nil
}
