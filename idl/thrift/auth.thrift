include "base.thrift"

namespace go auth

struct Subject {
    1: i64 UserId,
    2: string Username,
    3: string Role,
    4: map<string,string> Attributes, // e.g. department, clearance, tenant
}

struct LoginRequest {
    1: base.Base Base,
    2: string Username,
    3: string Password,
}

struct LoginResponse {
    1: string Token,
    2: i64 ExpiresAt, // unix seconds
    3: Subject Subject,
    255: base.BaseResp BaseResp,
}

struct ValidateTokenRequest {
    1: base.Base Base,
    2: string Token,
}

struct ValidateTokenResponse {
    1: bool Valid,
    2: Subject Subject,
    255: base.BaseResp BaseResp,
}

// AuthorizeRequest is a standard ABAC decision request:
// "can Subject perform Action on Resource, given Context?"
struct AuthorizeRequest {
    1: base.Base Base,
    2: Subject Subject,
    3: string Resource,               // e.g. "post"
    4: string Action,                 // e.g. "read", "write", "delete", "admin"
    5: optional map<string,string> ResourceAttributes, // e.g. owner_id
    6: optional map<string,string> Context,            // e.g. ip, time_of_day
}

struct AuthorizeResponse {
    1: bool Allowed,
    2: string Reason,
    3: string MatchedPolicy,
    255: base.BaseResp BaseResp,
}

// AuthService is the ABAC policy decision point (PDP) plus token issuance.
// It is a separate RPC service so any backend (gateway, postsvc, future
// services) can call a single, consistently-enforced authorization source.
service AuthService {
    LoginResponse Login(1: LoginRequest req),
    ValidateTokenResponse ValidateToken(1: ValidateTokenRequest req),
    AuthorizeResponse Authorize(1: AuthorizeRequest req),
}
