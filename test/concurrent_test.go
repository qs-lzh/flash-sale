package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/qs-lzh/flash-sale/config"
	"github.com/qs-lzh/flash-sale/internal/cache"
	"github.com/qs-lzh/flash-sale/internal/model"
)

const baseURL = "http://127.0.0.1:4000"

type ReserveRequest struct {
	UserID     uint `json:"user_id"`
	ShowtimeID uint `json:"showtime_id"`
}

type TestResult struct {
	SuccessCount    int64
	SoldOutCount    int64
	AlreadyOrdered  int64
	OtherErrorCount int64
	TotalRequests   int64
	TotalDuration   time.Duration
	AvgResponseTime time.Duration
}

func setupTestDB(t *testing.T, userCount, showtimeCount, ticketCount int) *gorm.DB {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// clear and rebuild tables
	db.Migrator().DropTable(&model.Order{}, &model.Showtime{}, &model.Movie{}, &model.User{})
	db.Migrator().AutoMigrate(&model.User{}, &model.Movie{}, &model.Showtime{}, &model.Order{})

	for i := 1; i <= userCount; i++ {
		user := model.User{
			Name:           fmt.Sprintf("用户%d", i),
			HashedPassword: fmt.Sprintf("pass%d", i),
			Role:           model.RoleUser,
		}
		db.Create(&user)
	}

	movie := model.Movie{
		Title:       "流浪地球3",
		Description: "科幻电影",
	}
	db.Create(&movie)

	for i := 1; i <= showtimeCount; i++ {
		showtime := model.Showtime{
			MovieID: 1,
			StartAt: time.Now().Add(time.Duration(i*2) * time.Hour),
		}
		db.Create(&showtime)
	}

	redisCache, err := cache.NewRedisCache(cfg.CacheURL)
	if err != nil {
		t.Fatalf("Failed to create redis cache: %v", err)
	}

	showtimeIDTicketsMap := make(map[uint]int)
	for i := 1; i <= showtimeCount; i++ {
		showtimeIDTicketsMap[uint(i)] = ticketCount
	}

	if err := redisCache.Init(showtimeIDTicketsMap); err != nil {
		t.Fatalf("Failed to init redis cache: %v", err)
	}

	t.Logf("✅ 测试数据初始化完成: %d个用户, %d个场次, 每场%d张票", userCount, showtimeCount, ticketCount)

	return db
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        20000,
		MaxIdleConnsPerHost: 20000,
		MaxConnsPerHost:     20000,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	},
	Timeout: 5 * time.Second,
}

func sendReserveRequest(userID, showtimeID uint) (statusCode int, responseBody string, duration time.Duration, err error) {
	reqBody := ReserveRequest{
		UserID:     userID,
		ShowtimeID: showtimeID,
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest(
		"POST",
		baseURL+"/reserve",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return 0, "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := httpClient.Do(req)
	duration = time.Since(start)

	if err != nil {
		return 0, "", duration, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), duration, nil
}

func concurrentTest(t *testing.T, concurrency int, showtimeID uint, userIDGenerator func(int) uint) *TestResult {
	result := &TestResult{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalDuration int64

	startTime := time.Now()

	for i := range concurrency {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			userID := userIDGenerator(index)
			statusCode, body, duration, err := sendReserveRequest(userID, showtimeID)

			mu.Lock()
			defer mu.Unlock()

			atomic.AddInt64(&totalDuration, int64(duration))
			atomic.AddInt64(&result.TotalRequests, 1)

			if err != nil {
				atomic.AddInt64(&result.OtherErrorCount, 1)
				t.Logf("❌ 请求错误 [用户%d]: %v", userID, err)
				return
			}

			switch statusCode {
			case 200:
				atomic.AddInt64(&result.SuccessCount, 1)
			case 409:
				if contains(body, "sold out") {
					atomic.AddInt64(&result.SoldOutCount, 1)
				} else if contains(body, "Already ordered") {
					atomic.AddInt64(&result.AlreadyOrdered, 1)
				} else {
					atomic.AddInt64(&result.OtherErrorCount, 1)
					t.Logf("⚠️  409但非预期错误 [用户%d]: %s", userID, body)
				}
			default:
				atomic.AddInt64(&result.OtherErrorCount, 1)
				t.Logf("⚠️  未预期状态码 [用户%d]: %d, 响应: %s", userID, statusCode, body)
			}
		}(i)
	}

	wg.Wait()
	result.TotalDuration = time.Since(startTime)
	result.AvgResponseTime = time.Duration(totalDuration / result.TotalRequests)

	return result
}

func contains(s, substr string) bool {
	if s == substr {
		return true
	}
	return strings.Contains(s, substr)
}

func printTestResult(t *testing.T, scenarioName string, result *TestResult) {
	separator := stringHelper("=").repeat(60)
	t.Logf("\n%s", separator)
	t.Logf("📊 %s - 测试结果", scenarioName)
	t.Logf("%s", separator)
	t.Logf("✅ 成功预订: %d", result.SuccessCount)
	t.Logf("🔴 已售罄: %d", result.SoldOutCount)
	t.Logf("🔁 重复预订: %d", result.AlreadyOrdered)
	t.Logf("❌ 其他错误: %d", result.OtherErrorCount)
	t.Logf("📈 总请求数: %d", result.TotalRequests)
	t.Logf("⏱️  总耗时: %v", result.TotalDuration)
	t.Logf("⚡ 平均响应时间: %v", result.AvgResponseTime)
	t.Logf("🚀 QPS: %.2f", float64(result.TotalRequests)/result.TotalDuration.Seconds())
	t.Logf("%s\n", separator)
}

func verifyOrderCount(t *testing.T, db *gorm.DB, showtimeID uint, expectedCount int64) {
	var actualCount int64
	db.Model(&model.Order{}).Where("showtime_id = ?", showtimeID).Count(&actualCount)

	if actualCount != expectedCount {
		t.Errorf("❌ 数据库订单数不一致！期望: %d, 实际: %d", expectedCount, actualCount)
	} else {
		t.Logf("✅ 数据库验证通过: %d 条订单", actualCount)
	}
}

// 场景1: 极限抢票测试（超卖验证）
func TestConcurrent_OversellPrevention(t *testing.T) {
	const (
		ticketCount = 100  // 可配置：票数
		concurrency = 7000 // 可配置：并发数
		showtimeID  = 1
	)

	db := setupTestDB(t, concurrency, 1, ticketCount)

	t.Logf("\n🎯 场景1: 极限抢票测试")
	t.Logf("票数: %d, 并发用户: %d", ticketCount, concurrency)

	result := concurrentTest(t, concurrency, showtimeID, func(i int) uint {
		return uint(i + 1) // 每个goroutine使用不同的用户ID
	})

	printTestResult(t, "场景1: 超卖测试", result)

	// 验证：成功数应该等于票数
	if result.SuccessCount != ticketCount {
		t.Errorf("❌ 超卖检测失败！成功预订: %d, 实际票数: %d", result.SuccessCount, ticketCount)
	} else {
		t.Logf("✅ 超卖检测通过！")
	}

	// 验证：失败数应该等于并发数-票数
	expectedFailed := int64(concurrency - ticketCount)
	actualFailed := result.SoldOutCount + result.OtherErrorCount
	if actualFailed != expectedFailed {
		t.Errorf("❌ 失败数不符！期望: %d, 实际: %d", expectedFailed, actualFailed)
	}

	fmt.Printf("订票已完成，等待3秒保证数据库写入完成\n")
	time.Sleep(3 * time.Second)
	verifyOrderCount(t, db, showtimeID, ticketCount)
}

// 场景2: 同一用户幂等性测试
func TestConcurrent_IdempotencyCheck(t *testing.T) {
	const (
		concurrency = 20 // 可配置：并发数
		showtimeID  = 1
		userID      = 1
	)

	db := setupTestDB(t, 10, 1, 10)

	t.Logf("\n🎯 场景2: 同一用户幂等性测试")
	t.Logf("用户%d 发起 %d 个并发请求", userID, concurrency)

	result := concurrentTest(t, concurrency, showtimeID, func(i int) uint {
		return userID // 所有goroutine使用相同用户ID
	})

	printTestResult(t, "场景2: 幂等性测试", result)

	// 验证：只有1个成功
	if result.SuccessCount != 1 {
		t.Errorf("❌ 幂等性检测失败！成功次数: %d, 期望: 1", result.SuccessCount)
	} else {
		t.Logf("✅ 幂等性检测通过！")
	}

	// 验证：其余全部是"已预订"错误
	if result.AlreadyOrdered != int64(concurrency-1) {
		t.Errorf("❌ 重复预订错误数不符！期望: %d, 实际: %d", concurrency-1, result.AlreadyOrdered)
	}

	fmt.Printf("订票已完成，等待3秒保证数据库写入完成\n")
	time.Sleep(3 * time.Second)

	verifyOrderCount(t, db, showtimeID, 1)
}

// 场景3: 多场次混合测试
func TestConcurrent_MultipleShowtimes(t *testing.T) {
	const (
		showtimeCount      = 3    // 场次数
		ticketsPerShowtime = 50   // 每场票数
		totalConcurrency   = 3000 // 可配置：总并发数
	)

	db := setupTestDB(t, totalConcurrency, showtimeCount, ticketsPerShowtime)

	t.Logf("\n🎯 场景3: 多场次混合测试")
	t.Logf("%d个场次, 每场%d张票, 总并发: %d", showtimeCount, ticketsPerShowtime, totalConcurrency)

	var wg sync.WaitGroup
	results := make([]*TestResult, showtimeCount)

	// 对每个场次启动并发测试
	for showtimeID := 1; showtimeID <= showtimeCount; showtimeID++ {
		wg.Add(1)
		go func(sid int) {
			defer wg.Done()

			// 每个场次分配部分并发
			concurrency := totalConcurrency / showtimeCount
			userOffset := (sid - 1) * concurrency

			results[sid-1] = concurrentTest(t, concurrency, uint(sid), func(i int) uint {
				return uint(userOffset + i + 1)
			})
		}(showtimeID)
	}

	wg.Wait()

	fmt.Printf("订票已完成，等待3秒保证数据库写入完成\n")
	time.Sleep(3 * time.Second)

	// 汇总结果
	totalSuccess := int64(0)
	for i, result := range results {
		showtimeID := i + 1
		printTestResult(t, fmt.Sprintf("场景3-场次%d", showtimeID), result)
		totalSuccess += result.SuccessCount

		// 验证每个场次的订单数
		verifyOrderCount(t, db, uint(showtimeID), result.SuccessCount)
	}

	t.Logf("\n📊 多场次总结: 总成功预订 %d 笔", totalSuccess)
}

// 修复string扩展方法
type stringHelper string

func (s stringHelper) repeat(count int) string {
	result := ""
	for _ = range count {
		result += string(s)
	}
	return result
}
