namespace go base

// Base is embedded in every request to carry cross-cutting context
// (tracing, caller identity) across the RPC boundary.
struct Base {
    1: string LogID = "",
    2: string Caller = "",
    3: string Addr = "",
    4: string Client = "",
    5: optional map<string,string> Extra,
}

// BaseResp is embedded in every response so callers get a uniform
// status envelope regardless of which RPC was invoked.
struct BaseResp {
    1: string StatusMessage = "",
    2: i32 StatusCode = 0,
    3: optional map<string,string> Extra,
}
