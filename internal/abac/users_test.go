package abac

import "testing"

func TestUserStore_DemoAccountsAuthenticate(t *testing.T) {
	store := NewUserStore()

	cases := []struct {
		username, password, wantRole string
	}{
		{"editor", "editor123", "editor"},
		{"viewer", "viewer123", "viewer"},
	}
	for _, c := range cases {
		subj, ok := store.Authenticate(c.username, c.password)
		if !ok {
			t.Errorf("expected %s to authenticate", c.username)
			continue
		}
		if subj.Role != c.wantRole {
			t.Errorf("%s: expected role %s, got %s", c.username, c.wantRole, subj.Role)
		}
	}
}

func TestUserStore_WrongPasswordRejected(t *testing.T) {
	store := NewUserStore()
	if _, ok := store.Authenticate("editor", "wrong-password"); ok {
		t.Error("expected wrong password to be rejected")
	}
}

func TestUserStore_UnknownUserRejected(t *testing.T) {
	store := NewUserStore()
	if _, ok := store.Authenticate("nobody", "irrelevant"); ok {
		t.Error("expected unknown username to be rejected")
	}
}

func TestUserStore_AddUser(t *testing.T) {
	store := NewUserStore()
	store.AddUser("admin", "s3cret!", Subject{UserID: 1, Role: "admin"})

	subj, ok := store.Authenticate("admin", "s3cret!")
	if !ok {
		t.Fatal("expected the newly added admin user to authenticate")
	}
	if subj.Role != "admin" || subj.Username != "admin" {
		t.Errorf("unexpected subject: %+v", subj)
	}
}
