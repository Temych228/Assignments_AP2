package http

import (
	"net/http"
	"order-service/internal/domain"

	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase *usecase.OrderUsecase
}

func NewHandler(u *usecase.OrderUsecase) *Handler {
	return &Handler{usecase: u}
}

type createOrderRequest struct {
	CustomerID    string `json:"customer_id"`
	CustomerEmail string `json:"customer_email"`
	ItemName      string `json:"item_name"`
	Amount        int64  `json:"amount"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	type result struct {
		order *domain.Order // нужен импорт "order-service/internal/domain"
		err   error
	}

	const attempts = 5
	results := make(chan result, attempts)

	for i := 0; i < attempts; i++ {
		go func() {
			order, err := h.usecase.CreateOrder(
				req.CustomerID, req.CustomerEmail, req.ItemName, req.Amount, idempotencyKey,
			)
			results <- result{order, err}
		}()
	}

	var successOrder *domain.Order
	var lastErr error

	for i := 0; i < attempts; i++ {
		r := <-results
		if r.err == nil && r.order != nil && successOrder == nil {
			successOrder = r.order
		}
		if r.err != nil {
			lastErr = r.err
		}
	}

	if successOrder != nil {
		c.JSON(http.StatusOK, successOrder)
		return
	}

	switch lastErr {
	case usecase.ErrAmountMustBePositive:
		c.JSON(http.StatusBadRequest, gin.H{"error": lastErr.Error()})
	case usecase.ErrPaymentServiceDown:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": lastErr.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": lastErr.Error()})
	}
}

func (h *Handler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.usecase.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) GetOrderStats(c *gin.Context) {
	stats, err := h.usecase.GetOrderStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.usecase.CancelOrder(id)
	if err != nil {
		switch err {
		case usecase.ErrCannotCancelPaidOrder, usecase.ErrCannotCancelNonPendingOrder:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}
