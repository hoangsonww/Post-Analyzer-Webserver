include "base.thrift"

namespace go post

struct Post {
    1: i64 Id,
    2: i64 UserId,
    3: string Title,
    4: string Body,
    5: string CreatedAt, // RFC3339
    6: string UpdatedAt, // RFC3339
}

struct PostFilter {
    1: optional i64 UserId,
    2: optional string Search,
    3: optional string SortBy,    // id, title, createdAt, updatedAt
    4: optional string SortOrder, // asc, desc
}

struct Pagination {
    1: i32 Page,
    2: i32 PageSize,
}

struct PaginationMeta {
    1: i32 Page,
    2: i32 PageSize,
    3: i32 TotalItems,
    4: i32 TotalPages,
    5: bool HasNext,
    6: bool HasPrev,
}

struct ListPostsRequest {
    1: base.Base Base,
    2: optional PostFilter Filter,
    3: optional Pagination Pagination,
}

struct ListPostsResponse {
    1: list<Post> Posts,
    2: PaginationMeta Pagination,
    255: base.BaseResp BaseResp,
}

struct GetPostRequest {
    1: base.Base Base,
    2: i64 Id,
}

struct GetPostResponse {
    1: Post Post,
    255: base.BaseResp BaseResp,
}

struct CreatePostRequest {
    1: base.Base Base,
    2: i64 UserId,
    3: string Title,
    4: string Body,
}

struct CreatePostResponse {
    1: Post Post,
    255: base.BaseResp BaseResp,
}

struct UpdatePostRequest {
    1: base.Base Base,
    2: i64 Id,
    3: optional string Title,
    4: optional string Body,
}

struct UpdatePostResponse {
    1: Post Post,
    255: base.BaseResp BaseResp,
}

struct DeletePostRequest {
    1: base.Base Base,
    2: i64 Id,
}

struct DeletePostResponse {
    255: base.BaseResp BaseResp,
}

struct BatchCreatePostsRequest {
    1: base.Base Base,
    2: list<CreatePostRequest> Posts,
}

struct BatchCreatePostsResponse {
    1: i32 Created,
    2: i32 Failed,
    3: list<string> Errors,
    4: list<i64> PostIds,
    255: base.BaseResp BaseResp,
}

struct CharacterStat {
    1: string Character,
    2: i64 Count,
    3: double Frequency,
}

struct AnalyticsStats {
    1: double AveragePostLength,
    2: i32 MedianPostLength,
    3: map<i64,i32> PostsPerUser,
    4: map<string,i32> TimeDistribution,
}

struct AnalyzePostsRequest {
    1: base.Base Base,
}

struct AnalyzePostsResponse {
    1: i32 TotalPosts,
    2: i64 TotalCharacters,
    3: i32 UniqueChars,
    4: list<CharacterStat> TopCharacters,
    5: AnalyticsStats Statistics,
    255: base.BaseResp BaseResp,
}

// PostService owns post CRUD and character-frequency analysis. It is the
// RPC backend behind the HTTP gateway (idl/thrift comment: transport is
// Kitex over TTHeader+Netpoll; see cmd/postsvc).
service PostService {
    ListPostsResponse ListPosts(1: ListPostsRequest req),
    GetPostResponse GetPost(1: GetPostRequest req),
    CreatePostResponse CreatePost(1: CreatePostRequest req),
    UpdatePostResponse UpdatePost(1: UpdatePostRequest req),
    DeletePostResponse DeletePost(1: DeletePostRequest req),
    BatchCreatePostsResponse BatchCreatePosts(1: BatchCreatePostsRequest req),
    AnalyzePostsResponse AnalyzePosts(1: AnalyzePostsRequest req),
}
