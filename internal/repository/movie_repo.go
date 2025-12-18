package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/qs-lzh/flash-sale/internal/model"
)

type MovieRepo interface {
	WithTx(tx *gorm.DB) MovieRepo
	Create(movie *model.Movie) error
	GetByID(id uint) (*model.Movie, error)
	GetByTitle(title string) (*model.Movie, error)
	ListAll() ([]model.Movie, error)
}

type movieRepoGorm struct {
	db *gorm.DB
}

var _ MovieRepo = (*movieRepoGorm)(nil)

func NewMovieRepoGorm(db *gorm.DB) *movieRepoGorm {
	return &movieRepoGorm{
		db: db,
	}
}

func (r *movieRepoGorm) WithTx(tx *gorm.DB) MovieRepo {
	return &movieRepoGorm{
		db: tx,
	}
}

func (r *movieRepoGorm) Create(movie *model.Movie) error {
	ctx := context.Background()
	if err := gorm.G[model.Movie](r.db).Create(ctx, movie); err != nil {
		return err
	}
	return nil
}

func (r *movieRepoGorm) GetByID(id uint) (*model.Movie, error) {
	ctx := context.Background()
	movie, err := gorm.G[model.Movie](r.db).Where(&model.Movie{ID: id}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &movie, nil
}

func (r *movieRepoGorm) GetByTitle(title string) (*model.Movie, error) {
	ctx := context.Background()
	movie, err := gorm.G[model.Movie](r.db).Where(&model.Movie{Title: title}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &movie, nil
}

func (r *movieRepoGorm) ListAll() ([]model.Movie, error) {
	ctx := context.Background()
	movies, err := gorm.G[model.Movie](r.db).Find(ctx)
	if err != nil {
		return nil, err
	}
	return movies, nil
}
