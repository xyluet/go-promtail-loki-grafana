package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	constraintsext "github.com/go-playground/pkg/v5/constraints"
	"github.com/mroth/weightedrand/v2"
	"github.com/oklog/run"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(http.MethodGet, 2),
		weightedrand.NewChoice(http.MethodPost, 2),
		weightedrand.NewChoice(http.MethodDelete, 2),
		weightedrand.NewChoice(http.MethodPut, 2),
		weightedrand.NewChoice(http.MethodPatch, 2),
	)
	durChooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(generator(time.Millisecond, 2*time.Millisecond), 1),
		// weightedrand.NewChoice(generator(50*time.Millisecond, 100*time.Millisecond), 9),
		// weightedrand.NewChoice(generator(500*time.Millisecond, time.Second), 1),
	)
	_ = durChooser

	statusChooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(func() int { return http.StatusOK }, 8),
		weightedrand.NewChoice(generator(400, 404), 1),
		weightedrand.NewChoice(generator(500, 504), 1),
	)

	httpDurChooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(generator(50*time.Millisecond, 100*time.Millisecond), 995),
		weightedrand.NewChoice(generator(500*time.Millisecond, 600*time.Millisecond), 5),
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			logger.Info("Hello, world!",
				zap.String("tag", "traffic"),
				zap.String("http.method", chooser.Pick()),
				zap.Int("http.status", statusChooser.Pick()()),
				zap.Int64("http.duration", httpDurChooser.Pick()().Milliseconds()),
			)

			picker := durChooser.Pick()
			sleepDur := picker()
			time.Sleep(sleepDur)
		}
	}()

	var group run.Group
	group.Add(
		func() error {
			wg.Wait()
			return nil
		},
		func(err error) {
			cancel()
		},
	)
	group.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGTERM))
	fmt.Println(group.Run())
}

func generator[T constraintsext.Number](min, max T) func() T {
	return func() T { return T(generateRandomBetween(int64(min), int64(max))) }
}

func generateRandomBetween(min, max int64) int64 {
	randomDuration := rand.Int63n(max-min) + min
	return randomDuration
}
