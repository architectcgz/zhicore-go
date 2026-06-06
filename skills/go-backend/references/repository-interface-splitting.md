# Repository Interface Splitting

When a repository interface starts collecting methods for multiple use cases, split it by consumer need instead of keeping one provider-owned "god repository".

## Rule

- Do not keep a wide repository just because one concrete infrastructure type happens to implement all methods.
- Put the final interface composition in the consuming application package.
- Keep `ports` or `contracts` focused on small capability interfaces.
- Split transaction runners by use case when different write paths need different tx capabilities.

## Bad

```go
package ports

type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	List(ctx context.Context, filter UserListFilter) ([]*model.User, int64, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, userID int64) error
	UpdatePassword(ctx context.Context, userID int64, newHash string) error
	UpdateLoginState(ctx context.Context, userID int64, failedAttempts int, lastFailedAt, lockedUntil *time.Time, status string) error
	UpdateProfile(ctx context.Context, user *model.User) error
}
```

Problem:

- admin query, admin write, auth login, CAS sync, and profile update are unrelated consumers
- every service now depends on methods it does not own
- later transaction splitting becomes awkward because the boundary is already too broad

## Better

Small capability interfaces stay in `contracts` or `ports`:

```go
package contracts

type UserListRepository interface {
	List(ctx context.Context, filter UserListFilter) ([]*model.User, int64, error)
}

type UserLookupRepository interface {
	FindByID(ctx context.Context, userID int64) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
}

type UserWriteRepository interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, userID int64) error
}

type UserPasswordRepository interface {
	UpdatePassword(ctx context.Context, userID int64, newHash string) error
}

type UserLoginStateRepository interface {
	UpdateLoginState(ctx context.Context, userID int64, failedAttempts int, lastFailedAt, lockedUntil *time.Time, status string) error
}

type UserProfileRepository interface {
	UpdateProfile(ctx context.Context, user *model.User) error
}
```

Application composes only what it uses:

```go
package commands

type authUserRepository interface {
	contracts.UserLookupRepository
	contracts.UserWriteRepository
	contracts.UserLoginStateRepository
}

type service struct {
	users authUserRepository
}
```

```go
package queries

type adminQueryRepository interface {
	contracts.UserListRepository
}

type AdminService struct {
	repo adminQueryRepository
}
```

## Transaction Variant

If one write path needs a transaction, do not pass the same wide repository into the tx closure.

Bad:

```go
type SubmissionRepository interface {
	WithinTransaction(ctx context.Context, fn func(repo SubmissionRepository) error) error
	CreateSubmission(ctx context.Context, submission *model.Submission) error
	UpdateScore(ctx context.Context, submissionID int64, score int) error
	AddTeamScore(ctx context.Context, teamID int64, delta int) error
	FindRegistration(ctx context.Context, contestID, userID int64) (*model.ContestRegistration, error)
	FindChallengeByID(ctx context.Context, challengeID int64) (*model.Challenge, error)
}
```

Better:

```go
type SubmissionScoringTxRepository interface {
	CreateSubmission(ctx context.Context, submission *model.Submission) error
	UpdateScore(ctx context.Context, submissionID int64, score int) error
	AddTeamScore(ctx context.Context, teamID int64, delta int) error
}

type SubmissionScoringTxRunner interface {
	WithinScoringTransaction(ctx context.Context, fn func(repo SubmissionScoringTxRepository) error) error
}
```

## Review Checklist

- Does this interface mix query and command concerns from different consumers?
- Does one service depend on methods it never calls?
- Is the composition owned by infrastructure instead of the application consumer?
- Does a transaction closure receive more capability than the use case actually needs?

If any answer is yes, split the repository before continuing.
