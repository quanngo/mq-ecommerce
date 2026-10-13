package middleware

import (
	"main/component/repositories"
	"main/utils"
)

type ApiMiddleware struct {
	cfg       *utils.Config
	tokenRepo *repositories.TokenRepo
}

func NewMiddleware(cfg *utils.Config, tokenRepo *repositories.TokenRepo) *ApiMiddleware {
	return &ApiMiddleware{
		cfg:       cfg,
		tokenRepo: tokenRepo,
	}
}
