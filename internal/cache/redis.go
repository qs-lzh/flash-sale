package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type RedisCache struct {
	Client *redis.Client
}

func NewRedisCache(url string) (*RedisCache, error) {
	client := redis.NewClient(
		&redis.Options{
			Addr:     url,
			Password: "",
			DB:       0,
		},
	)
	redisCache := &RedisCache{Client: client}

	return redisCache, nil
}

func (r *RedisCache) Init(showtimeIDTicketsMap map[uint]int) error {
	if err := r.Client.FlushDB(context.Background()).Err(); err != nil {
		return err
	}
	if err := r.initRemainingTickets(showtimeIDTicketsMap); err != nil {
		return err
	}
	return nil
}

func (r *RedisCache) initRemainingTickets(showtimeIDTicketsMap map[uint]int) error {
	args := make([]any, 0, len(showtimeIDTicketsMap)*2)
	for showtimeID, tickets := range showtimeIDTicketsMap {
		key := MakeShowtimeRemainingTicketsKey(showtimeID) // "showtime:{id}:ticket:remain"
		args = append(args, key, tickets)
	}

	_, err := initTicketsScript.Run(ctx, r.Client, []string{}, args...).Result()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisCache) Set(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisCache) Get(key string, dest any) error {
	data, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (r *RedisCache) SetBool(key string, value bool) error {
	strValue := "false"
	if value {
		strValue = "true"
	}
	return r.Client.Set(ctx, key, strValue, 5*time.Minute).Err()
}

func (r *RedisCache) GetBool(key string) (value bool, err error) {
	value, err = r.Client.Get(ctx, key).Bool()
	if err != nil {
		return false, err
	}
	return value, nil
}

/*
* remaining tickets of a showtime
 */

// create a reservation in redis if there's tickets available
func (r *RedisCache) ReserveTicket(showtimeID uint, userID uint) (reservationID uint, err error) {
	remainingTicketsKey := MakeShowtimeRemainingTicketsKey(showtimeID)
	userShowtimeOrderedKey := MakeUserShowtimeOrderedKey(userID, showtimeID)
	res, err := reserveTicketScript.Run(ctx, r.Client, []string{remainingTicketsKey, ReservationIDSeqKey, userShowtimeOrderedKey}, showtimeID, userID).Int64()
	if err != nil {
		return 0, err
	}
	if res == -1 {
		return 0, ErrSoldOut
	}
	if res == -3 {
		return 0, ErrAlreadyOrdered
	}

	reservationID = uint(res)

	return reservationID, nil
}

func (r *RedisCache) MarkTicketAsPaid(reservationID uint) error {
	res, err := markTicketAsPaidScript.Run(ctx, r.Client, []string{fmt.Sprintf("reservation:%d", reservationID)}).Result()
	if err != nil {
		return err
	}
	if res == int64(-2) {
		return errors.New("invalid reservation status")
	}
	return nil
}

// mark ticket as timeout and roll back remaining tickets in redis
func (r *RedisCache) MarkTicketAsTimeout(reservationID uint) error {
	res, err := markTicketAsTimeoutScript.Run(ctx, r.Client, []string{fmt.Sprintf("reservation:%d", reservationID)}).Result()
	if err != nil {
		return err
	}
	if res == int64(-2) {
		return errors.New("invalid reservation status")
	}
	return nil
}

func (r *RedisCache) ReleaseTicket(showtimeID uint) error {
	key := MakeShowtimeRemainingTicketsKey(showtimeID)
	return r.Client.Incr(ctx, key).Err()
}

/*
* user ordered showtime
 */
func (r *RedisCache) SetOrdered(userID uint, showtimeID uint) error {
	key := MakeUserShowtimeOrderedKey(userID, showtimeID)
	return r.Client.Set(ctx, key, true, 0).Err()
}

func (r *RedisCache) GetOrdered(userID uint, showtimeID uint) (bool, error) {
	key := MakeUserShowtimeOrderedKey(userID, showtimeID)
	exist, err := r.Client.Get(ctx, key).Bool()
	if err != nil {
		// if the user doesn't order the showtime
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		// internal error
		return false, err
	}
	// if the user doesn't order the showtime
	return exist, nil
}

func (r *RedisCache) GetReservationInfo(reservationID uint) (map[string]string, error) {
	key := MakeReservationKey(reservationID)
	return r.Client.HGetAll(ctx, key).Result()
}
