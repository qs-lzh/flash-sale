package domain

import (
	"errors"

	"github.com/qs-lzh/flash-sale/internal/cache"
)

type ReservationService interface {
	Reserve(userID, showtimeID uint) (orderID uint, err error)
}

type reservationService struct {
	Cache *cache.RedisCache
}

func NewReservationService(cache *cache.RedisCache) *reservationService {
	return &reservationService{
		Cache: cache,
	}
}

var _ ReservationService = (*reservationService)(nil)

func (s *reservationService) Reserve(userID, showtimeID uint) (orderID uint, err error) {
	orderID, err = s.Cache.ReserveTicket(showtimeID, userID)
	if err != nil {
		if errors.Is(err, cache.ErrSoldOut) {
			return 0, cache.ErrSoldOut
		}
		if errors.Is(err, cache.ErrAlreadyOrdered) {
			return 0, cache.ErrAlreadyOrdered
		}
		return 0, err
	}
	return orderID, nil
}
