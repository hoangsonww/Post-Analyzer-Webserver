# Post Analyzer API Documentation

**Version:** 2.0.0
**Base URL:** `http://localhost:8080/api/v1`

## Overview

The Post Analyzer API provides a comprehensive RESTful interface for managing and analyzing posts. The API features include:

- ✅ Full CRUD operations for posts
- ✅ Advanced filtering and pagination
- ✅ Bulk operations
- ✅ Character frequency analytics
- ✅ Data export (JSON/CSV)
- ✅ Request validation and error handling
- ✅ Rate limiting and security

## Authentication

Currently, the API is open and does not require authentication. Authentication will be added in a future release.

## Rate Limiting

- **Default Limit:** 100 requests per minute per IP
- **Headers:** Rate limit status is available in response headers
- **429 Response:** Returns when limit is exceeded

## Base Response Format

### Success Response
```json
{
  "data": { /* response data */ },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

### Error Response
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "fields": {
      "fieldName": "Field-specific error"
    }
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

## Endpoints

### 1. List Posts

Retrieve a paginated list of posts with optional filtering and sorting.

**Endpoint:** `GET /api/v1/posts`

**Query Parameters:**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `page` | integer | Page number | 1 |
| `pageSize` | integer | Items per page (max 100) | 20 |
| `userId` | integer | Filter by user ID | - |
| `search` | string | Search in title and body | - |
| `sortBy` | string | Sort field (id, title, createdAt, updatedAt) | id |
| `sortOrder` | string | Sort order (asc, desc) | desc |

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/posts?page=1&pageSize=10&sortBy=createdAt&sortOrder=desc"
```

**Example Response:**
```json
{
  "data": [
    {
      "id": 1,
      "userId": 1,
      "title": "Sample Post",
      "body": "This is the post content...",
      "createdAt": "2025-01-16T10:00:00Z",
      "updatedAt": "2025-01-16T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "pageSize": 10,
    "totalItems": 100,
    "totalPages": 10,
    "hasNext": true,
    "hasPrev": false
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

### 2. Get Post by ID

Retrieve a single post by its ID.

**Endpoint:** `GET /api/v1/posts/{id}`

**Path Parameters:**
- `id` (integer, required): Post ID

**Example Request:**
```bash
curl http://localhost:8080/api/v1/posts/1
```

**Example Response:**
```json
{
  "data": {
    "id": 1,
    "userId": 1,
    "title": "Sample Post",
    "body": "This is the post content...",
    "createdAt": "2025-01-16T10:00:00Z",
    "updatedAt": "2025-01-16T10:00:00Z"
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

**Error Response (404):**
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Post not found"
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

### 3. Create Post

Create a new post.

**Endpoint:** `POST /api/v1/posts`

**Request Body:**
```json
{
  "userId": 1,
  "title": "My New Post",
  "body": "This is the content of my new post."
}
```

**Validation Rules:**
- `title`: Required, 1-500 characters
- `body`: Required, 1-10,000 characters
- `userId`: Optional, defaults to 1

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/v1/posts \
  -H "Content-Type: application/json" \
  -d '{
    "userId": 1,
    "title": "My New Post",
    "body": "This is the content of my new post."
  }'
```

**Example Response (201 Created):**
```json
{
  "data": {
    "id": 101,
    "userId": 1,
    "title": "My New Post",
    "body": "This is the content of my new post.",
    "createdAt": "2025-01-16T10:30:00Z",
    "updatedAt": "2025-01-16T10:30:00Z"
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

**Validation Error Response (422):**
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "validation failed",
    "fields": {
      "title": "title is required",
      "body": "body too long (max 10000 characters)"
    }
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:30:00Z"
  }
}
```

### 4. Update Post

Update an existing post.

**Endpoint:** `PUT /api/v1/posts/{id}`

**Path Parameters:**
- `id` (integer, required): Post ID

**Request Body:**
```json
{
  "title": "Updated Title",
  "body": "Updated content..."
}
```

**Note:** All fields are optional. Only provided fields will be updated.

**Example Request:**
```bash
curl -X PUT http://localhost:8080/api/v1/posts/1 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Title",
    "body": "Updated content..."
  }'
```

**Example Response:**
```json
{
  "data": {
    "id": 1,
    "userId": 1,
    "title": "Updated Title",
    "body": "Updated content...",
    "createdAt": "2025-01-16T10:00:00Z",
    "updatedAt": "2025-01-16T10:35:00Z"
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:35:00Z"
  }
}
```

### 5. Delete Post

Delete a post by ID.

**Endpoint:** `DELETE /api/v1/posts/{id}`

**Path Parameters:**
- `id` (integer, required): Post ID

**Example Request:**
```bash
curl -X DELETE http://localhost:8080/api/v1/posts/1
```

**Example Response:**
```json
{
  "message": "Post deleted successfully",
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:40:00Z"
  }
}
```

### 6. Bulk Create Posts

Create multiple posts in a single request.

**Endpoint:** `POST /api/v1/posts/bulk`

**Request Body:**
```json
{
  "posts": [
    {
      "userId": 1,
      "title": "First Post",
      "body": "Content of first post"
    },
    {
      "userId": 1,
      "title": "Second Post",
      "body": "Content of second post"
    }
  ]
}
```

**Limits:**
- Minimum: 1 post
- Maximum: 1000 posts per request

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/v1/posts/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "posts": [
      {"userId": 1, "title": "Post 1", "body": "Content 1"},
      {"userId": 1, "title": "Post 2", "body": "Content 2"}
    ]
  }'
```

**Example Response (201/207):**
```json
{
  "data": {
    "created": 2,
    "failed": 0,
    "errors": [],
    "postIds": [102, 103]
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:45:00Z"
  }
}
```

**Partial Success (207 Multi-Status):**
```json
{
  "data": {
    "created": 1,
    "failed": 1,
    "errors": [
      "post 2: validation failed"
    ],
    "postIds": [102]
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:45:00Z"
  }
}
```

### 7. Export Posts

Export posts in JSON or CSV format.

**Endpoint:** `GET /api/v1/posts/export`

**Query Parameters:**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `format` | string | Export format (json, csv) | json |
| `userId` | integer | Filter by user ID | - |
| `search` | string | Search in title and body | - |

**Example Request (JSON):**
```bash
curl "http://localhost:8080/api/v1/posts/export?format=json" \
  -o posts_export.json
```

**Example Request (CSV):**
```bash
curl "http://localhost:8080/api/v1/posts/export?format=csv" \
  -o posts_export.csv
```

**JSON Response:**
```json
[
  {
    "id": 1,
    "userId": 1,
    "title": "Sample Post",
    "body": "Content...",
    "createdAt": "2025-01-16T10:00:00Z",
    "updatedAt": "2025-01-16T10:00:00Z"
  }
]
```

**CSV Response:**
```csv
ID,UserID,Title,Body,CreatedAt,UpdatedAt
1,1,"Sample Post","Content...","2025-01-16T10:00:00Z","2025-01-16T10:00:00Z"
```

### 8. Analyze Posts

Perform character frequency analysis on all posts.

**Endpoint:** `GET /api/v1/posts/analytics`

**Example Request:**
```bash
curl http://localhost:8080/api/v1/posts/analytics
```

**Example Response:**
```json
{
  "data": {
    "totalPosts": 100,
    "totalCharacters": 50000,
    "uniqueChars": 95,
    "charFrequency": {
      "32": 8500,
      "97": 4200,
      "101": 6500
    },
    "topCharacters": [
      {
        "character": " ",
        "count": 8500,
        "frequency": 17.0
      },
      {
        "character": "e",
        "count": 6500,
        "frequency": 13.0
      }
    ],
    "statistics": {
      "averagePostLength": 500.0,
      "medianPostLength": 475,
      "postsPerUser": {
        "1": 50,
        "2": 30
      },
      "timeDistribution": {
        "morning": 25,
        "afternoon": 40,
        "evening": 30,
        "night": 5
      }
    }
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2025-01-16T10:50:00Z",
    "duration": "250ms"
  }
}
```

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `NOT_FOUND` | 404 | Resource not found |
| `INVALID_INPUT` | 400 | Invalid input data |
| `VALIDATION_FAILED` | 422 | Validation failed |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `CONFLICT` | 409 | Resource conflict |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service unavailable |

## Pagination

All list endpoints support pagination with the following parameters:

- `page`: Page number (starts at 1)
- `pageSize`: Number of items per page (max 100, default 20)

**Example:**
```bash
curl "http://localhost:8080/api/v1/posts?page=2&pageSize=25"
```

## Filtering

List endpoints support various filters:

- `userId`: Filter by user ID
- `search`: Full-text search in title and body

**Example:**
```bash
curl "http://localhost:8080/api/v1/posts?userId=1&search=golang"
```

## Sorting

List endpoints support sorting with:

- `sortBy`: Field to sort by (id, title, createdAt, updatedAt)
- `sortOrder`: Sort direction (asc, desc)

**Example:**
```bash
curl "http://localhost:8080/api/v1/posts?sortBy=createdAt&sortOrder=desc"
```

## Code Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

// Create a post
async function createPost() {
  try {
    const response = await axios.post('http://localhost:8080/api/v1/posts', {
      userId: 1,
      title: 'My Post',
      body: 'Post content...'
    });
    console.log('Created:', response.data);
  } catch (error) {
    console.error('Error:', error.response.data);
  }
}

// List posts with pagination
async function listPosts(page = 1) {
  const response = await axios.get('http://localhost:8080/api/v1/posts', {
    params: { page, pageSize: 20 }
  });
  return response.data;
}
```

### Python

```python
import requests

BASE_URL = 'http://localhost:8080/api/v1'

# Create a post
def create_post():
    response = requests.post(f'{BASE_URL}/posts', json={
        'userId': 1,
        'title': 'My Post',
        'body': 'Post content...'
    })
    return response.json()

# Get posts with filtering
def get_posts(user_id=None, search=None):
    params = {}
    if user_id:
        params['userId'] = user_id
    if search:
        params['search'] = search

    response = requests.get(f'{BASE_URL}/posts', params=params)
    return response.json()

# Analyze posts
def analyze_posts():
    response = requests.get(f'{BASE_URL}/posts/analytics')
    return response.json()
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const baseURL = "http://localhost:8080/api/v1"

type Post struct {
    ID     int    `json:"id,omitempty"`
    UserID int    `json:"userId"`
    Title  string `json:"title"`
    Body   string `json:"body"`
}

func createPost(post Post) (*Post, error) {
    data, _ := json.Marshal(post)
    resp, err := http.Post(baseURL+"/posts", "application/json", bytes.NewBuffer(data))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data Post `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    return &result.Data, nil
}
```

### cURL Examples

```bash
# Create a post
curl -X POST http://localhost:8080/api/v1/posts \
  -H "Content-Type: application/json" \
  -d '{"userId":1,"title":"Test","body":"Content"}'

# Get all posts
curl http://localhost:8080/api/v1/posts

# Get post by ID
curl http://localhost:8080/api/v1/posts/1

# Update post
curl -X PUT http://localhost:8080/api/v1/posts/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title"}'

# Delete post
curl -X DELETE http://localhost:8080/api/v1/posts/1

# Search posts
curl "http://localhost:8080/api/v1/posts?search=golang&sortBy=createdAt&sortOrder=desc"

# Export to CSV
curl "http://localhost:8080/api/v1/posts/export?format=csv" -o export.csv

# Get analytics
curl http://localhost:8080/api/v1/posts/analytics
```

## Versioning

The API uses URL path versioning. Current version is `v1`.

- **v1 Endpoints:** `/api/v1/*`
- **Default:** `/api/*` routes to v1

## Best Practices

1. **Always include Content-Type header** for POST/PUT requests
2. **Handle pagination** for large datasets
3. **Use bulk endpoints** for multiple operations
4. **Implement retry logic** with exponential backoff
5. **Cache responses** where appropriate
6. **Monitor rate limits** in response headers
7. **Validate input** before sending requests
8. **Handle errors gracefully** with proper error messages

## Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642348800
```

## Support

For issues and feature requests, please visit:
https://github.com/hoangsonww/Post-Analyzer-Webserver/issues

---

**Last Updated:** January 16, 2025
**API Version:** 2.0.0
