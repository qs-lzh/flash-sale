package app

import (
	"github.com/qs-lzh/flash-sale/config"
	"github.com/qs-lzh/flash-sale/internal/cache"
	"github.com/qs-lzh/flash-sale/internal/mq"
	"github.com/qs-lzh/flash-sale/internal/repository"
	"github.com/qs-lzh/flash-sale/internal/service/domain"
	"github.com/qs-lzh/flash-sale/internal/service/workflow"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config *config.Config

	DB     *gorm.DB
	Cache  *cache.RedisCache
	Logger *zap.Logger
	MQConn *amqp.Connection

	UserRepo     *repository.UserRepo
	MovieRepo    *repository.MovieRepo
	ShowtimeRepo *repository.ShowtimeRepo

	MovieService       domain.MovieService
	ShowtimeService    domain.ShowtimeService
	ReservationService domain.ReservationService
	OrderService       domain.OrderService
	PaymentService     domain.PaymentService

	ReservationWorkflow *workflow.ReservationWorkflow
	PaymentWorkflow     *workflow.PaymentWorkflow
	OrderWorkflow       *workflow.OrderWorkflow
}

func New(config *config.Config, db *gorm.DB, cache *cache.RedisCache, mqConn *amqp.Connection) *App {
	movieRepo := repository.NewMovieRepoGorm(db)
	showtimeRepo := repository.NewShowtimeRepoGorm(db)
	orderRepo := repository.NewOrderRepoGorm(db)

	showtimeService := domain.NewShowtimeService(db, showtimeRepo)
	reservationService := domain.NewReservationService(cache)
	orderService := domain.NewOrderService(db, cache, orderRepo)
	movieService := domain.NewMovieService(db, movieRepo, showtimeService)
	paymentService := domain.NewPaymentService(cache)

	reservationWorkflow := workflow.NewReservationWorkflow(reservationService, mqConn)
	paymentWorkflow := workflow.NewPaymentWorkflow(paymentService, mqConn)
	orderWorkflow := workflow.NewOrderWorkflow(cache, orderService)

	return &App{
		Config:              config,
		DB:                  db,
		Cache:               cache,
		MQConn:              mqConn,
		MovieService:        movieService,
		ShowtimeService:     showtimeService,
		ReservationService:  reservationService,
		OrderService:        orderService,
		PaymentService:      paymentService,
		ReservationWorkflow: reservationWorkflow,
		PaymentWorkflow:     paymentWorkflow,
		OrderWorkflow:       orderWorkflow,
	}
}

func (app *App) Init() error {
	// init redis
	showtimeIDTicketsMap := make(map[uint]int)
	showtimes, err := app.ShowtimeService.GetAllShowtimes()
	if err != nil {
		return err
	}
	for _, showtime := range showtimes {
		showtimeIDTicketsMap[showtime.ID] = 100 // 100 tickets
	}
	if err := app.Cache.Init(showtimeIDTicketsMap); err != nil {
		return err
	}

	// init rabbit mq
	mq.InitQueues(app.MQConn)

	app.PaymentWorkflow.Start(app.MQConn)
	app.OrderWorkflow.Start(app.MQConn)

	return nil
}

func (app *App) Close() error {
	sqlDB, err := app.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
