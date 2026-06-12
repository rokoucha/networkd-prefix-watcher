package app

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/rokoucha/networkd-prefix-watcher/internal/config"
	"github.com/rokoucha/networkd-prefix-watcher/internal/detector"
	"github.com/rokoucha/networkd-prefix-watcher/internal/envfile"
	"github.com/rokoucha/networkd-prefix-watcher/internal/prefix"
	"github.com/rokoucha/networkd-prefix-watcher/internal/state"
	"github.com/rokoucha/networkd-prefix-watcher/internal/trigger"
)

func Run(ctx context.Context, cfg config.Config, logger *log.Logger) error {
	d, err := detector.New(cfg)
	if err != nil {
		return err
	}
	defer d.Close()
	return runWith(ctx, cfg, logger, d, trigger.SystemctlRunner{})
}

func runWith(ctx context.Context, cfg config.Config, logger *log.Logger, d detector.Detector, runner trigger.Runner) error {
	current, err := d.Snapshot(ctx)
	if err != nil {
		return err
	}
	if err := reconcile(ctx, cfg, logger, runner, current); err != nil {
		return err
	}
	if cfg.Once {
		return nil
	}

	events := make(chan error, 1)
	go func() {
		<-ctx.Done()
		_ = d.Close()
	}()
	go func() {
		for {
			if err := d.Wait(ctx); err != nil {
				if ctx.Err() != nil {
					events <- ctx.Err()
					return
				}
				events <- err
				return
			}
			select {
			case events <- nil:
			case <-ctx.Done():
				return
			}
		}
	}()

	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-events:
			if err != nil {
				return err
			}
			if cfg.Debounce == 0 {
				current, err := d.Snapshot(ctx)
				if err != nil {
					return err
				}
				if err := reconcile(ctx, cfg, logger, runner, current); err != nil {
					return err
				}
				continue
			}
			if timer == nil {
				timer = time.NewTimer(cfg.Debounce)
				timerC = timer.C
				continue
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(cfg.Debounce)
			timerC = timer.C
		case <-timerC:
			timerC = nil
			current, err := d.Snapshot(ctx)
			if err != nil {
				return err
			}
			if err := reconcile(ctx, cfg, logger, runner, current); err != nil {
				return err
			}
		}
	}
}

func reconcile(ctx context.Context, cfg config.Config, logger *log.Logger, runner trigger.Runner, current prefix.Set) error {
	current = prefix.Normalize(current)
	old, err := state.Load(cfg.StateFile)
	first := errors.Is(err, state.ErrNotFound)
	if err != nil && !first {
		if errors.Is(err, os.ErrPermission) {
			return err
		}
		logger.Printf("ignoring unreadable state file %s: %v", cfg.StateFile, err)
		first = true
	}

	previous := prefix.Set(nil)
	if !first {
		previous, err = old.PrefixSet()
		if err != nil {
			logger.Printf("ignoring corrupt state file %s: %v", cfg.StateFile, err)
			first = true
		}
	}

	if first {
		if err := state.Save(cfg.StateFile, state.New(cfg.Mode, cfg.Interface, cfg.PrefixLen, current)); err != nil {
			return err
		}
		if !cfg.TriggerOnStart {
			logger.Printf("recorded initial prefixes: %s", current.Join())
			return nil
		}
	} else if current.Equal(previous) {
		return nil
	}

	if err := envfile.Write(cfg.EnvFile, current, previous); err != nil {
		return err
	}
	if err := state.Save(cfg.StateFile, state.New(cfg.Mode, cfg.Interface, cfg.PrefixLen, current)); err != nil {
		return err
	}
	if err := runner.RestartTarget(ctx, cfg.Target); err != nil {
		return err
	}
	logger.Printf("triggered %s: %q -> %q", cfg.Target, previous.Join(), current.Join())
	return nil
}
