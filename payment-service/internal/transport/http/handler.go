package http

import (
	"net/http"

	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase *usecase.PaymentUsecase
}

func NewHandler(u *usecase.PaymentUsecase) *Handler {
	return &Handler{usecase: u}
}

type createPaymentRequest struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.usecase.ProcessPayment(c.Request.Context(), req.OrderID, req.Amount, req.CustomerEmail)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, payment)
}

func (h *Handler) GetPayment(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := h.usecase.GetPayment(orderID)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}

	c.JSON(200, payment)
}
