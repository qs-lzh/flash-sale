package cache

import (
	"errors"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

// key names definition
// key names in lua script should follow these formats
const (
	ReservationKey      = "reservation:%d"     // key of reservation details, '%d' is reservation id
	ReservationIDSeqKey = "reservation:id:seq" // reservation id, a constant

	ShowtimeRemainingTicketsKey = "showtime:%d:ticket:remain" // key of remaining tickets of a showtime, '%d' is showtime id

	UserShowtimeOrderedKey = "user:%d:showtime:%d:ordered" // key of a user's reservation to a showtime, first '%d' is user id, second '%d' is showtime id
)

func MakeReservationKey(reservationID uint) string {
	return fmt.Sprintf("reservation:%d", reservationID)
}

func MakeShowtimeRemainingTicketsKey(showtimeID uint) string {
	return fmt.Sprintf("showtime:%d:ticket:remain", showtimeID)
}

func MakeUserShowtimeOrderedKey(userID uint, showtimeID uint) string {
	return fmt.Sprintf("user:%d:showtime:%d:ordered", userID, showtimeID)
}

// struct definitions
// the data put into redis in lua script should follow the struct
type ReservationCacheValue struct {
	ShowtimeID uint              `redis:"showtime_id"`
	SeatID     uint              `redis:"seat_id"`
	UserID     uint              `redis:"user_id"`
	Status     ReservationStatus `redis:"status"`
}

type ReservationStatus string

var (
	ReservationStatusReserved ReservationStatus = "RESERVED"
	ReservationStatusPaid     ReservationStatus = "PAID"
	ReservationStatusTimeout  ReservationStatus = "TIMEOUT"
)

// errors
var (
	ErrSoldOut        = errors.New("Tickets sold out")
	ErrAlreadyOrdered = errors.New("User already ordered this showtime")
)

// lua scripts
var initTicketsScript = redis.NewScript(`
-- ARGV: key1 value1 key2 value2 ...
for i = 1, #ARGV, 2 do
    local key = ARGV[i]
    local value = tonumber(ARGV[i + 1])
    redis.call("SET", key, value)
end
return #ARGV / 2
`)

var reserveTicketScript = redis.NewScript(`
	-- KEYS[1] = showtime:{showtime_id}:ticket:remain
	-- KEYS[2] = reservation:id:seq
	-- KEYS[3] = user:{user_id}:showtime:{showtime_id}:ordered

	-- ARGV[1] = showtime_id
	-- ARGV[2] = user_id

	-- 检查用户是否已经订过该场次的票
	local userOrderedKey = KEYS[3]
	local hasOrdered = redis.call("GET", userOrderedKey)
	if hasOrdered then
		return -3  -- 表示用户已订单
	end

	-- 检查剩余票数
	local remain = tonumber(redis.call("GET", KEYS[1]))
	if (not remain) or remain <= 0 then
		return -1  -- 表示售罄
	end

	-- 扣库存
	redis.call("DECR", KEYS[1])

	-- 生成 reservation_id
	local id = redis.call("INCR", KEYS[2])

	local resKey = "reservation:" .. id

	-- 创建 reservation
	redis.call("HSET", resKey,
		"showtime_id", ARGV[1],
		"user_id", ARGV[2],
		"status", "RESERVED"
	)

	-- 标记用户已订单 (无过期时间，永久有效)
	redis.call("SET", userOrderedKey, "true")

	return id
`)

var markTicketAsPaidScript = redis.NewScript(`
	-- KEYS[1] = reservation:{reservation_id}

	local resKey = KEYS[1]
	local status = redis.call("HGET", resKey, "status")
	if not status or status ~= "RESERVED" then
		return -2
	end

	redis.call("HSET", resKey, "status", "PAID")
	return 1
`)

var markTicketAsTimeoutScript = redis.NewScript(`
	-- KEYS[1] = reservation:{reservation_id}

	local resKey = KEYS[1]
	local status = redis.call("HGET", resKey, "status")
	local showtime_id = redis.call("HGET", resKey, "showtime_id")

	if not status or status ~= "RESERVED" then
		return -2
	end

	-- 构建库存键
	local remainKey = "showtime:" .. showtime_id .. ":ticket:remain"

	-- 更新状态为超时
	redis.call("HSET", resKey, "status", "TIMEOUT")

	-- 增加对应场次的剩余票数
	redis.call("INCR", remainKey)

	return 1
`)
