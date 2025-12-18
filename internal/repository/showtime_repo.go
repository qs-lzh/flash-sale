package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/qs-lzh/flash-sale/internal/model"
)

type ShowtimeRepo interface {
	WithTx(tx *gorm.DB) ShowtimeRepo
	Create(showtime *model.Showtime) error
	GetByID(id uint) (*model.Showtime, error)
	GetByMovieID(movieID uint) ([]model.Showtime, error)
	ListAll() ([]model.Showtime, error)
}

type showtimeRepoGorm struct {
	db *gorm.DB
}

var _ ShowtimeRepo = (*showtimeRepoGorm)(nil)

func NewShowtimeRepoGorm(db *gorm.DB) *showtimeRepoGorm {
	return &showtimeRepoGorm{
		db: db,
	}
}

func (r *showtimeRepoGorm) WithTx(tx *gorm.DB) ShowtimeRepo {
	return &showtimeRepoGorm{
		db: tx,
	}
}

func (r *showtimeRepoGorm) Create(showtime *model.Showtime) error {
	ctx := context.Background()
	if err := gorm.G[model.Showtime](r.db).Create(ctx, showtime); err != nil {
		return err
	}
	return nil
}

func (r *showtimeRepoGorm) GetByID(id uint) (*model.Showtime, error) {
	ctx := context.Background()
	showtime, err := gorm.G[model.Showtime](r.db).Where(&model.Showtime{ID: id}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &showtime, nil
}

func (r *showtimeRepoGorm) GetByMovieID(movieID uint) ([]model.Showtime, error) {
	ctx := context.Background()
	showtimes, err := gorm.G[model.Showtime](r.db).Where(&model.Showtime{MovieID: movieID}).Find(ctx)
	if err != nil {
		return nil, err
	}
	return showtimes, nil
}

func (r *showtimeRepoGorm) ListAll() ([]model.Showtime, error) {
	ctx := context.Background()
	showtimes, err := gorm.G[model.Showtime](r.db).Find(ctx)
	if err != nil {
		return nil, err
	}
	return showtimes, nil
}
