package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

type AuthHandler struct {
	authService AuthService
	validator   *RequestValidator
}

func RegisterAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthHandler, error) {
		return &AuthHandler{
			authService: do.MustInvoke[*service.AuthService](i),
			validator:   do.MustInvoke[*RequestValidator](i),
		}, nil
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req api.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	dtoReq := dto.RegisterRequest{
		Email:    string(req.Email),
		Username: req.Username,
		Password: req.Password,
		Name:     req.Name,
	}
	if !h.validator.Validate(c, dtoReq) {
		return
	}

	ctx := c.Request.Context()
	result, err := h.authService.Register(ctx, dtoReq)
	if err != nil {
		if errors.Is(err, service.ErrEmailOrUsernameTaken) {
			c.JSON(http.StatusConflict, errResponse("CONFLICT", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, api.AuthResponse{
		Token: result.Token,
		User: api.UserProfile{
			Id:       int(result.User.ID),
			Email:    result.User.Email,
			Username: result.User.Username,
			Name:     result.User.Name,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req api.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	dtoReq := dto.LoginRequest{
		Login:    req.Login,
		Password: req.Password,
	}
	if !h.validator.Validate(c, dtoReq) {
		return
	}

	ctx := c.Request.Context()
	result, err := h.authService.Login(ctx, dtoReq)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.AuthResponse{
		Token: result.Token,
		User: api.UserProfile{
			Id:       int(result.User.ID),
			Email:    result.User.Email,
			Username: result.User.Username,
			Name:     result.User.Name,
		},
	})
}
