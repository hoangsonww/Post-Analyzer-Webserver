// Package abac implements the policy decision point (PDP) for
// attribute-based access control: given a subject (who), a resource +
// action (what), and attributes on the resource/request context (which
// conditions hold), decide allow or deny. It is transport-agnostic —
// authsvc exposes it over Kitex RPC (idl/thrift/auth.thrift), and the
// gateway middleware calls that RPC per request.
package abac

// Subject is the authenticated caller attempting an action.
type Subject struct {
	UserID     int
	Username   string
	Role       string // admin | editor | viewer
	Attributes map[string]string
}

// Request is a single ABAC decision request: "can Subject perform Action
// on Resource, given ResourceAttributes and Context?"
type Request struct {
	Subject            Subject
	Resource           string
	Action             string
	ResourceAttributes map[string]string
	Context            map[string]string
}

// Decision is the PDP's answer.
type Decision struct {
	Allowed       bool
	Reason        string
	MatchedPolicy string
}

// Effect is what a matched policy does.
type Effect string

const (
	Allow Effect = "allow"
	Deny  Effect = "deny"
)

// Policy is one ABAC rule. Condition may be nil (matches unconditionally
// once Resource/Action/Role match); when set, it inspects request/context
// attributes, which is what makes this attribute-based rather than plain
// role-based access control.
type Policy struct {
	Name      string
	Resource  string // "*" matches any resource
	Action    string // "*" matches any action
	Role      string // "*" matches any role
	Effect    Effect
	Condition func(Request) bool
}

func (p Policy) matches(req Request) bool {
	if p.Resource != "*" && p.Resource != req.Resource {
		return false
	}
	if p.Action != "*" && p.Action != req.Action {
		return false
	}
	if p.Role != "*" && p.Role != req.Subject.Role {
		return false
	}
	if p.Condition != nil && !p.Condition(req) {
		return false
	}
	return true
}

// DefaultPolicies is the built-in policy set for this service's one
// protected resource ("post"). Evaluated top to bottom; first match wins.
// Everything not explicitly allowed is denied (default-deny).
func DefaultPolicies() []Policy {
	return []Policy{
		{
			Name: "admin-full-access", Resource: "*", Action: "*", Role: "admin",
			Effect: Allow,
		},
		{
			// Attribute-based: deleting a post additionally requires an
			// "mfa" context attribute, even for editors — demonstrating a
			// condition beyond plain role membership.
			Name: "editor-delete-requires-mfa", Resource: "post", Action: "delete", Role: "editor",
			Effect: Allow,
			Condition: func(r Request) bool {
				return r.Context["mfa"] == "true"
			},
		},
		{
			Name: "editor-write", Resource: "post", Action: "write", Role: "editor",
			Effect: Allow,
		},
		{
			Name: "editor-read", Resource: "post", Action: "read", Role: "editor",
			Effect: Allow,
		},
		{
			Name: "viewer-read", Resource: "post", Action: "read", Role: "viewer",
			Effect: Allow,
		},
	}
}

// Evaluate runs req through policies in order and returns the first
// match, or a default-deny Decision if nothing matches.
func Evaluate(req Request, policies []Policy) Decision {
	for _, p := range policies {
		if p.matches(req) {
			return Decision{
				Allowed:       p.Effect == Allow,
				Reason:        string(p.Effect) + " by policy " + p.Name,
				MatchedPolicy: p.Name,
			}
		}
	}
	return Decision{Allowed: false, Reason: "no matching policy (default-deny)", MatchedPolicy: ""}
}
