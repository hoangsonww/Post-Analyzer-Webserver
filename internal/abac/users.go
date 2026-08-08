package abac

import "golang.org/x/crypto/bcrypt"

type storedUser struct {
	passwordHash string
	subject      Subject
}

// UserStore is an in-memory credential store for authsvc. It is seeded
// with demo editor/viewer accounts at startup; the admin account comes
// from AuthConfig (env-configurable) via AddUser so it isn't hardcoded.
type UserStore struct {
	users map[string]storedUser
}

func NewUserStore() *UserStore {
	s := &UserStore{users: make(map[string]storedUser)}
	// Demo accounts illustrating the role hierarchy. Local/dev only —
	// documented in README, not meant for production use as-is.
	s.AddUser("editor", "editor123", Subject{UserID: 2, Username: "editor", Role: "editor"})
	s.AddUser("viewer", "viewer123", Subject{UserID: 3, Username: "viewer", Role: "viewer"})
	return s
}

func (s *UserStore) AddUser(username, password string, subj Subject) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err) // only fails on invalid cost/oversized input, never at startup with fixed constants
	}
	subj.Username = username
	s.users[username] = storedUser{passwordHash: string(hash), subject: subj}
}

func (s *UserStore) Authenticate(username, password string) (Subject, bool) {
	u, ok := s.users[username]
	if !ok {
		return Subject{}, false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password)) != nil {
		return Subject{}, false
	}
	return u.subject, true
}
