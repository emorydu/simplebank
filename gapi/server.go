package gapi

import (
	"fmt"

	db "github.com/emorydu/simplebank/db/sqlc"
	"github.com/emorydu/simplebank/pb"
	"github.com/emorydu/simplebank/token"
	"github.com/emorydu/simplebank/util"
	"github.com/emorydu/simplebank/worker"
)

// Server serves gRPC requests for out banking service.
type Server struct {
	pb.UnimplementedSimpleBankServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

// NewServer create a new gRPC server.
func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}
