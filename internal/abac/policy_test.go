package abac

import "testing"

func TestEvaluate_AdminAllowsEverything(t *testing.T) {
	policies := DefaultPolicies()
	req := Request{
		Subject:  Subject{Role: "admin"},
		Resource: "post",
		Action:   "delete",
	}
	d := Evaluate(req, policies)
	if !d.Allowed {
		t.Fatalf("expected admin to be allowed, got denied: %s", d.Reason)
	}
	if d.MatchedPolicy != "admin-full-access" {
		t.Errorf("expected admin-full-access to match, got %q", d.MatchedPolicy)
	}
}

func TestEvaluate_ViewerReadOnly(t *testing.T) {
	policies := DefaultPolicies()

	allow := Evaluate(Request{Subject: Subject{Role: "viewer"}, Resource: "post", Action: "read"}, policies)
	if !allow.Allowed {
		t.Errorf("expected viewer read to be allowed, got denied: %s", allow.Reason)
	}

	deny := Evaluate(Request{Subject: Subject{Role: "viewer"}, Resource: "post", Action: "write"}, policies)
	if deny.Allowed {
		t.Errorf("expected viewer write to be denied")
	}
	if deny.MatchedPolicy != "" {
		t.Errorf("expected no policy match for denied viewer write, got %q", deny.MatchedPolicy)
	}
}

func TestEvaluate_EditorWriteButNotDelete(t *testing.T) {
	policies := DefaultPolicies()

	write := Evaluate(Request{Subject: Subject{Role: "editor"}, Resource: "post", Action: "write"}, policies)
	if !write.Allowed {
		t.Errorf("expected editor write to be allowed, got denied: %s", write.Reason)
	}

	deleteNoMFA := Evaluate(Request{
		Subject: Subject{Role: "editor"}, Resource: "post", Action: "delete",
		Context: map[string]string{"mfa": "false"},
	}, policies)
	if deleteNoMFA.Allowed {
		t.Errorf("expected editor delete without MFA to be denied")
	}
}

func TestEvaluate_EditorDeleteRequiresMFAAttribute(t *testing.T) {
	policies := DefaultPolicies()

	// This is the case that distinguishes ABAC from plain RBAC: same
	// role, same resource, same action — the only thing that changes is
	// a context attribute.
	withMFA := Evaluate(Request{
		Subject: Subject{Role: "editor"}, Resource: "post", Action: "delete",
		Context: map[string]string{"mfa": "true"},
	}, policies)
	if !withMFA.Allowed {
		t.Errorf("expected editor delete with mfa=true to be allowed, got denied: %s", withMFA.Reason)
	}

	withoutMFA := Evaluate(Request{
		Subject: Subject{Role: "editor"}, Resource: "post", Action: "delete",
		Context: map[string]string{},
	}, policies)
	if withoutMFA.Allowed {
		t.Errorf("expected editor delete without mfa context to be denied")
	}
}

func TestEvaluate_DefaultDenyForUnknownRole(t *testing.T) {
	policies := DefaultPolicies()
	d := Evaluate(Request{Subject: Subject{Role: "nobody"}, Resource: "post", Action: "read"}, policies)
	if d.Allowed {
		t.Errorf("expected unknown role to be denied by default")
	}
	if d.Reason == "" {
		t.Errorf("expected a reason to be given for the deny")
	}
}

func TestEvaluate_DefaultDenyForUnknownResource(t *testing.T) {
	policies := DefaultPolicies()
	// Non-admin roles only have policies scoped to resource "post" — an
	// unrelated resource name should fall through to default-deny even
	// for actions that would be allowed on "post".
	d := Evaluate(Request{Subject: Subject{Role: "editor"}, Resource: "admin", Action: "read"}, policies)
	if d.Allowed {
		t.Errorf("expected editor to be denied on resource 'admin'")
	}
}

func TestPolicyMatches_WildcardsAndCondition(t *testing.T) {
	calls := 0
	p := Policy{
		Name: "wildcard-with-condition", Resource: "*", Action: "*", Role: "*",
		Effect: Allow,
		Condition: func(r Request) bool {
			calls++
			return r.Context["flag"] == "on"
		},
	}

	on := p.matches(Request{Context: map[string]string{"flag": "on"}})
	off := p.matches(Request{Context: map[string]string{"flag": "off"}})

	if !on {
		t.Error("expected condition-true request to match")
	}
	if off {
		t.Error("expected condition-false request not to match")
	}
	if calls != 2 {
		t.Errorf("expected condition to be evaluated twice, got %d", calls)
	}
}
