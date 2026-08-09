package main

import (
	"context"

	"Post_Analyzer_Webserver/internal/adapt"
	"Post_Analyzer_Webserver/internal/models"
	"Post_Analyzer_Webserver/internal/service"
	post "Post_Analyzer_Webserver/kitex_gen/post"
)

// PostServiceImpl implements the Kitex-generated post.PostService interface,
// delegating all business logic to the shared internal/service.PostService
// (the same logic the HTTP gateway used before the RPC split). It also
// publishes lifecycle events to Kafka/RocketMQ via events (nil-safe).
type PostServiceImpl struct {
	svc    *service.PostService
	events *EventPublisher
}

func NewPostServiceImpl(svc *service.PostService, events *EventPublisher) *PostServiceImpl {
	return &PostServiceImpl{svc: svc, events: events}
}

func (s *PostServiceImpl) ListPosts(ctx context.Context, req *post.ListPostsRequest) (*post.ListPostsResponse, error) {
	filter := adapt.ThriftFilterToModel(req.Filter)
	pagination := adapt.ThriftPaginationToModel(req.Pagination)

	posts, meta, err := s.svc.GetAll(ctx, filter, pagination)
	if err != nil {
		return &post.ListPostsResponse{Pagination: &post.PaginationMeta{}, BaseResp: adapt.Err(err)}, nil
	}
	return &post.ListPostsResponse{
		Posts:      adapt.ModelsToThriftPosts(posts),
		Pagination: adapt.ModelPaginationToThrift(meta),
		BaseResp:   adapt.OK(),
	}, nil
}

func (s *PostServiceImpl) GetPost(ctx context.Context, req *post.GetPostRequest) (*post.GetPostResponse, error) {
	p, err := s.svc.GetByID(ctx, int(req.Id))
	if err != nil {
		return &post.GetPostResponse{BaseResp: adapt.Err(err)}, nil
	}
	return &post.GetPostResponse{Post: adapt.ModelToThriftPost(*p), BaseResp: adapt.OK()}, nil
}

func (s *PostServiceImpl) CreatePost(ctx context.Context, req *post.CreatePostRequest) (*post.CreatePostResponse, error) {
	p, err := s.svc.Create(ctx, &models.CreatePostRequest{
		UserID: int(req.UserId),
		Title:  req.Title,
		Body:   req.Body,
	})
	if err != nil {
		return &post.CreatePostResponse{BaseResp: adapt.Err(err)}, nil
	}
	s.events.PublishPostEvent(ctx, "created", *p)
	s.events.PublishScheduledRecheck(ctx, *p)
	return &post.CreatePostResponse{Post: adapt.ModelToThriftPost(*p), BaseResp: adapt.OK()}, nil
}

func (s *PostServiceImpl) UpdatePost(ctx context.Context, req *post.UpdatePostRequest) (*post.UpdatePostResponse, error) {
	upd := &models.UpdatePostRequest{}
	if req.Title != nil {
		upd.Title = *req.Title
	}
	if req.Body != nil {
		upd.Body = *req.Body
	}
	p, err := s.svc.Update(ctx, int(req.Id), upd)
	if err != nil {
		return &post.UpdatePostResponse{BaseResp: adapt.Err(err)}, nil
	}
	s.events.PublishPostEvent(ctx, "updated", *p)
	return &post.UpdatePostResponse{Post: adapt.ModelToThriftPost(*p), BaseResp: adapt.OK()}, nil
}

func (s *PostServiceImpl) DeletePost(ctx context.Context, req *post.DeletePostRequest) (*post.DeletePostResponse, error) {
	if err := s.svc.Delete(ctx, int(req.Id)); err != nil {
		return &post.DeletePostResponse{BaseResp: adapt.Err(err)}, nil
	}
	s.events.PublishPostEvent(ctx, "deleted", models.Post{ID: int(req.Id)})
	return &post.DeletePostResponse{BaseResp: adapt.OK()}, nil
}

func (s *PostServiceImpl) BatchCreatePosts(ctx context.Context, req *post.BatchCreatePostsRequest) (*post.BatchCreatePostsResponse, error) {
	bulkReq := &models.BulkCreateRequest{Posts: make([]models.CreatePostRequest, len(req.Posts))}
	for i, p := range req.Posts {
		bulkReq.Posts[i] = models.CreatePostRequest{UserID: int(p.UserId), Title: p.Title, Body: p.Body}
	}
	resp, err := s.svc.BulkCreate(ctx, bulkReq)
	if err != nil {
		return &post.BatchCreatePostsResponse{BaseResp: adapt.Err(err)}, nil
	}
	postIDs := make([]int64, len(resp.PostIDs))
	for i, id := range resp.PostIDs {
		postIDs[i] = int64(id)
	}
	return &post.BatchCreatePostsResponse{
		Created:  int32(resp.Created),
		Failed:   int32(resp.Failed),
		Errors:   resp.Errors,
		PostIds:  postIDs,
		BaseResp: adapt.OK(),
	}, nil
}

func (s *PostServiceImpl) AnalyzePosts(ctx context.Context, req *post.AnalyzePostsRequest) (*post.AnalyzePostsResponse, error) {
	result, err := s.svc.AnalyzeCharacterFrequency(ctx)
	if err != nil {
		return &post.AnalyzePostsResponse{BaseResp: adapt.Err(err)}, nil
	}

	topChars := make([]*post.CharacterStat, len(result.TopCharacters))
	for i, c := range result.TopCharacters {
		topChars[i] = &post.CharacterStat{
			Character: string(c.Character),
			Count:     int64(c.Count),
			Frequency: c.Frequency,
		}
	}

	stats := &post.AnalyticsStats{}
	if result.Statistics != nil {
		stats.AveragePostLength = result.Statistics.AveragePostLength
		stats.MedianPostLength = int32(result.Statistics.MedianPostLength)
		stats.PostsPerUser = make(map[int64]int32, len(result.Statistics.PostsPerUser))
		for uid, count := range result.Statistics.PostsPerUser {
			stats.PostsPerUser[int64(uid)] = int32(count)
		}
		stats.TimeDistribution = make(map[string]int32, len(result.Statistics.TimeDistribution))
		for k, v := range result.Statistics.TimeDistribution {
			stats.TimeDistribution[k] = int32(v)
		}
	}

	s.events.PublishPostEvent(ctx, "analyzed", models.Post{})

	return &post.AnalyzePostsResponse{
		TotalPosts:      int32(result.TotalPosts),
		TotalCharacters: int64(result.TotalCharacters),
		UniqueChars:     int32(result.UniqueChars),
		TopCharacters:   topChars,
		Statistics:      stats,
		BaseResp:        adapt.OK(),
	}, nil
}
