package port

type UnitOfWork interface {
	Do(fn func(uow UnitOfWork) error) error
	UserRepo() UserRepository
	OauthAccountRepo() OauthAccountRepository
	RoleRepo() RoleRepository
	SessionRepo() SessionRepository
}
