package repository

import (
	"context"

	"github.com/qs-lzh/flash-sale/internal/model"
	"gorm.io/gorm"
)

type UserRepo interface {
	WithTx(tx *gorm.DB) UserRepo
	Create(user *model.User) error
	GetByName(name string) (*model.User, error)
}

type userRepoGorm struct {
	db *gorm.DB
}

var _ UserRepo = (*userRepoGorm)(nil)

func NewUserRepoGorm(db *gorm.DB) *userRepoGorm {
	return &userRepoGorm{
		db: db,
	}
}

func (r *userRepoGorm) WithTx(tx *gorm.DB) UserRepo {
	return &userRepoGorm{
		db: tx,
	}
}

// default value of user.Role is 'user'
func (r *userRepoGorm) Create(user *model.User) error {
	ctx := context.Background()
	if err := gorm.G[model.User](r.db).Create(ctx, user); err != nil {
		return err
	}
	return nil
}

func (r *userRepoGorm) GetByName(name string) (*model.User, error) {
	ctx := context.Background()
	user, err := gorm.G[model.User](r.db).Where(model.User{Name: name}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
