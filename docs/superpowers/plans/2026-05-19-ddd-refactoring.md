# System Optimization Implementation Plan - Phase 1: DDD Refactoring

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Strictly decouple domain logic from infrastructure by moving password hashing to the User entity and enforcing entity mapping in repositories.

**Architecture:** DDD (Domain-Driven Design). Business logic resides in `internal/domain/entity`. Repositories in `internal/infrastructure/persistence` handle mapping between entities and ORM models.

**Tech Stack:** Go, GORM, Bcrypt.

---

### Task 1: Clean up User Entity and remove GORM hooks

**Files:**
- Modify: `internal/domain/entity/user.go`
- Modify: `internal/infrastructure/persistence/mysql/model/user.go`
- Test: `internal/domain/entity/entity_test.go`

- [ ] **Step 1: Ensure User entity has HashPassword and CheckPassword**

Confirm `internal/domain/entity/user.go` content:
```go
func HashPassword(raw string) (string, error) {
	if raw == "" {
		return "", ErrInvalidParam
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(raw, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(raw)) == nil
}
```

- [ ] **Step 2: Add test cases for password hashing in `internal/domain/entity/entity_test.go`**

```go
func TestUser_Password(t *testing.T) {
	password := "test123456"
	hashed, err := HashPassword(password)
	assert.Nil(t, err)
	assert.NotEqual(t, password, hashed)
	assert.True(t, CheckPassword(password, hashed))
	assert.False(t, CheckPassword("wrong", hashed))
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/domain/entity/...`
Expected: PASS

- [ ] **Step 4: Remove GORM hooks from `internal/infrastructure/persistence/mysql/model/user.go`**

Remove any `BeforeCreate` or `BeforeUpdate` methods if they exist. (Research showed `model/user.go` has a comment about hooks, need to ensure no code remains).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/entity/ internal/infrastructure/persistence/mysql/model/
git commit -m "refactor(domain): ensure user entity owns password logic and remove infra hooks"
```

---

### Task 2: Refactor User Repository to enforce Entity Mapping

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/userdb/user.go`

- [ ] **Step 1: Update CreateUser to accept entity.User**

```go
func (r *userRepoStruct) CreateUser(ctx context.Context, user *entity.User) error {
    m := &model.User{
        UserID:   user.UserID,
        UserName: user.UserName,
        Passwd:   user.Password,
        Role:     user.Role,
    }
    return r.db.WithContext(ctx).Create(m).Error
}
```

- [ ] **Step 2: Update GetUserByName and GetUserByID to return entity.User**

Ensure they map `model.User` to `entity.User` before returning.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/persistence/mysql/userdb/
git commit -m "refactor(infra): user repository now returns domain entities"
```

---

### Task 3: Refactor UserService to use Domain Logic

**Files:**
- Modify: `internal/application/user/user_service.go`

- [ ] **Step 1: Update SignUp to hash password manually**

```go
func (s *userServiceStruct) SignUp(ctx context.Context, p *userreq.SignUpRequest) error {
    // ...
    hashedPassword, err := entity.HashPassword(p.Password)
    if err != nil {
        return err
    }
    user := &entity.User{
        UserID:   snowflake.GenID(),
        UserName: p.Username,
        Password: hashedPassword,
        Role:     entity.RoleUser,
    }
    return s.userRepo.CreateUser(ctx, user)
}
```

- [ ] **Step 2: Update Login to use entity.CheckPassword**

- [ ] **Step 3: Commit**

```bash
git add internal/application/user/
git commit -m "refactor(app): user service now uses rich domain logic for passwords"
```
