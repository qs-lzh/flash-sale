package repository

import (
	"context"

	"github.com/qs-lzh/flash-sale/internal/model"
	"gorm.io/gorm"
)

type OrderRepo interface {
	WithTx(tx *gorm.DB) OrderRepo
	Create(order *model.Order) error
	GetByID(id uint) (*model.Order, error)
	GetByUserID(userID uint) ([]model.Order, error)
	GetByShowtimeID(showtimeID uint) ([]model.Order, error)
}

type orderRepoGorm struct {
	db *gorm.DB
}

var _ OrderRepo = (*orderRepoGorm)(nil)

func NewOrderRepoGorm(db *gorm.DB) *orderRepoGorm {
	return &orderRepoGorm{
		db: db,
	}
}

func (r *orderRepoGorm) WithTx(tx *gorm.DB) OrderRepo {
	return &orderRepoGorm{
		db: tx,
	}
}

func (r *orderRepoGorm) Create(order *model.Order) error {
	ctx := context.Background()
	if err := gorm.G[model.Order](r.db).Create(ctx, order); err != nil {
		return err
	}
	return nil
}

func (r *orderRepoGorm) GetByID(id uint) (*model.Order, error) {
	ctx := context.Background()
	order, err := gorm.G[model.Order](r.db).Where(&model.Order{ID: id}).First(ctx)
	if err != nil {
		return &model.Order{}, err
	}
	return &order, nil
}

func (r *orderRepoGorm) GetByUserID(userID uint) ([]model.Order, error) {
	ctx := context.Background()
	orders, err := gorm.G[model.Order](r.db).Where(&model.Order{UserID: userID}).Find(ctx)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *orderRepoGorm) GetByShowtimeID(showtimeID uint) ([]model.Order, error) {
	ctx := context.Background()
	orders, err := gorm.G[model.Order](r.db).Where(&model.Order{ShowtimeID: showtimeID}).Find(ctx)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
