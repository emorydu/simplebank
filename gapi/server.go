package gapi

import (
	db "github.com/emorydu/simplebank/db/sqlc"
	"github.com/emorydu/simplebank/token"
	"github.com/emorydu/simplebank/util"
	"github.com/emorydu/simplebank/worker"
)

// Server serves gRPC requests for out banking service.
type Server struct {
	// todo
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}
