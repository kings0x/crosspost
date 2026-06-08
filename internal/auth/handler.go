package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *AuthService
}

func NewAuthHandler(service *AuthService) *AuthHandler {
	return &AuthHandler{service}
}

func (h *AuthHandler) DoSomehting(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

//db would be
//users
//user_sessions
//user_tokens(one time stuff)
//i think a seperate place for oauth

//6 functions
//first is google oauth
//second is to handle the callback
//third is the signup function
//fourth is the login function
//fifth is the token refresh function
//sixth is the logout function

//then we would have two pure functions
//issue_token
//setupProvider
