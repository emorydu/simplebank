package api

import (
	"fmt"

	db "github.com/emorydu/simplebank/db/sqlc"
	"github.com/emorydu/simplebank/token"
	"github.com/emorydu/simplebank/util"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Server serves HTTP requests for our banking service.
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err = v.RegisterValidation("currency", validCurrency)
		if err != nil {
			return nil, fmt.Errorf("cannot register validation: %w", err)
		}
	}

	server.setupRouter()

	return server, nil
}

func (s *Server) setupRouter() {
	router := gin.Default()

	router.POST("/users", s.createUser)
	router.POST("/users/login", s.loginUser)
	router.POST("/tokens/renew_access", s.renewAccessToken)

	authRoutes := router.Group("/").Use(authMiddleware(s.tokenMaker))
	authRoutes.POST("/accounts", s.createAccount)
	authRoutes.GET("/accounts/:id", s.getAccount)
	authRoutes.GET("/accounts", s.listAccounts)

	authRoutes.POST("/transfers", s.createTransfer)

	s.router = router
}

// Start runs the HTTP server on a specific address.
func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
