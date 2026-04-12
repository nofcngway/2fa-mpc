package twofaService

//go:generate minimock -i Storage -o ./mocks/ -s _mock.go
//go:generate minimock -i SessionStorage -o ./mocks/ -s _mock.go

// Storage defines the interface for TwoFA persistent data access.
type Storage interface {
	// Methods added in Phase 7
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	// Methods added in Phase 7
}

// TwoFAService implements 2FA orchestration business logic.
type TwoFAService struct {
	storage        Storage
	sessionStorage SessionStorage
}

// NewTwoFAService creates a new TwoFAService instance.
func NewTwoFAService(storage Storage, sessionStorage SessionStorage) *TwoFAService {
	return &TwoFAService{
		storage:        storage,
		sessionStorage: sessionStorage,
	}
}
