package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/qs-lzh/flash-sale/internal/app"
	"github.com/qs-lzh/flash-sale/internal/cache"
)

type ReserveHandler struct {
	app *app.App
}

func NewReserveHandler(app *app.App) *ReserveHandler {
	return &ReserveHandler{
		app: app,
	}
}

func (h *ReserveHandler) HandleReserve(ctx *gin.Context) {
	var req ReserveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{
			"error":  "Invalid request format",
			"detail": err.Error(),
		})
		return
	}

	if err := h.app.ReservationWorkflow.Reserve(req.UserID, req.ShowtimeID); err != nil {
		if errors.Is(err, cache.ErrSoldOut) {
			ctx.JSON(409, gin.H{
				"error":   "Tickets sold out",
				"message": "Sorry, all tickets for this showtime have been sold out",
			})
			return
		}
		if errors.Is(err, cache.ErrAlreadyOrdered) {
			ctx.JSON(409, gin.H{
				"error":   "Already ordered",
				"message": "You have already reserved a ticket for this showtime",
			})
			return
		}
		ctx.JSON(500, gin.H{
			"error":   "Internal server error",
			"message": "Failed to process reservation, please try again later",
		})
		return
	}

	ctx.JSON(200, gin.H{
		"message": "Ticket reserved successfully",
		"status":  "RESERVED",
		"note":    "Please complete payment within 15 minutes",
	})
}

type ReserveRequest struct {
	UserID     uint `json:"user_id"`
	ShowtimeID uint `json:"showtime_id"`
}
