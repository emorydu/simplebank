package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/emorydu/simplebank/api"
	db "github.com/emorydu/simplebank/db/sqlc"
	"github.com/emorydu/simplebank/mail"
	"github.com/emorydu/simplebank/util"
	"github.com/emorydu/simplebank/worker"
	"github.com/golang-migrate/migrate/v4"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	runDBMigration(config.MigrationURL, config.DBSource)

	store := db.NewStore(connPool)

	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}

	//taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	waitGroup, ctx := errgroup.WithContext(ctx)
	runTaskProcessor(ctx, waitGroup, config, redisOpt, store)

	runGinServer(config, store)
	//runGatewayServer(ctx, waitGroup, config, store, taskDistributor)
	//runGrpcServer(ctx, waitGroup, config, store, taskDistributor)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("error from wait group")
	}
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}

//func runGatewayServer(
//	ctx context.Context,
//	waitGroup *errgroup.Group,
//	config util.Config,
//	store db.Store,
//	taskDistributor worker.TaskDistributor,
//) {
//	gapi.NewServer(config, store, taskDistributor)
//}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}

	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}

	log.Info().Msg("db migrated successfully")
}

func runTaskProcessor(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	redisOpt asynq.RedisClientOpt,
	store db.Store,
) {
	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)

	waitGroup.Go(func() error {
		log.Info().Msg("start task processor")
		err := taskProcessor.Start()
		if err != nil {
			log.Error().Err(err).Msg("failed to start task processor")
			return err
		}
		log.Info().Msg("task processor is stopped")
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown task processor")
		taskProcessor.Shutdown()
		return nil
	})
}
