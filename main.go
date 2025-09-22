package main

import (
	"context"
	"fmt"
	"github.com/tochemey/goakt/v3/discovery/static"
	"github.com/tochemey/goakt/v3/remote"
	"os"
	"os/signal"
	"syscall"

	goakt "github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"
	"github.com/tochemey/goakt/v3/log"
)

const (
	ActorsCount = 10000
	ClusterMode = true
)

type Actor1 struct {
	SelfID int
}

func NewActor1(selfID int) *Actor1 {
	return &Actor1{
		SelfID: selfID,
	}
}

func (n *Actor1) PreStart(_ *goakt.Context) (err error) {
	return nil
}

func (n *Actor1) Receive(ctx *goakt.ReceiveContext) {
	switch ctx.Message().(type) {
	case *goaktpb.PostStart:
	default:
		ctx.Unhandled()
	}
}

func (n *Actor1) PostStop(*goakt.Context) error {
	return nil
}

func main() {
	ctx := context.Background()
	logger := log.New(log.InfoLevel, os.Stdout)

	var actorSystem goakt.ActorSystem
	var err error
	if ClusterMode {
		staticConfig := static.Config{
			Hosts: []string{
				"localhost:3322",
			},
		}

		discovery := static.NewDiscovery(&staticConfig)

		clusterConfig := goakt.
			NewClusterConfig().
			WithDiscovery(discovery).
			WithPartitionCount(20).
			WithMinimumPeersQuorum(1).
			WithReplicaCount(1).
			WithDiscoveryPort(3322).
			WithPeersPort(3320).
			WithKinds(new(Actor1))

		actorSystem, err = goakt.NewActorSystem("memleak1",
			goakt.WithLogger(logger),
			goakt.WithActorInitMaxRetries(1),
			goakt.WithShutdownTimeout(10),
			goakt.WithRemote(remote.NewConfig("localhost", 50022)),
			goakt.WithCluster(clusterConfig),
		)
	} else {
		actorSystem, err = goakt.NewActorSystem("memleak1",
			goakt.WithLogger(logger),
			goakt.WithActorInitMaxRetries(1),
			goakt.WithShutdownTimeout(10),
		)
	}

	if err != nil {
		panic(err)
	}

	logger.Info("Start system")
	err = actorSystem.Start(ctx)
	if err != nil {
		panic(err)
	}
	logger.Info("System started")

	logger.Infof("Swawning %d actors", ActorsCount)

	for i := 0; i < ActorsCount; i++ {
		_, err = actorSystem.Spawn(
			context.Background(),
			fmt.Sprintf("actor-%d", i),
			NewActor1(i),
			goakt.WithLongLived(),
			goakt.WithSupervisor(
				goakt.NewSupervisor(
					goakt.WithStrategy(goakt.OneForOneStrategy),
					goakt.WithAnyErrorDirective(goakt.ResumeDirective),
				)),
		)
		if err != nil {
			panic(err)
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	signal.Notify(ch, syscall.SIGINT)

	<-ch

	err = actorSystem.Stop(ctx)
	logger.Info("System stopped")
}
