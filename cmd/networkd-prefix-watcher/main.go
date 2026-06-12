package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rokoucha/networkd-prefix-watcher/internal/app"
	"github.com/rokoucha/networkd-prefix-watcher/internal/config"
	"github.com/spf13/cobra"
)

func main() {
	cmd := newRootCommand(os.Stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "networkd-prefix-watcher: %v\n", err)
		os.Exit(2)
	}
}

func newRootCommand(stderr io.Writer) *cobra.Command {
	var cfg config.Config
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	cmd := &cobra.Command{
		Use:           "networkd-prefix-watcher",
		Short:         "Watch IPv6 prefix changes and trigger a systemd target",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer stop()
			if err := cfg.Validate(); err != nil {
				return err
			}
			if err := app.Run(ctx, cfg, log.New(stderr, "", log.LstdFlags)); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			}
			return nil
		},
	}
	cmd.SetErr(stderr)

	flags := cmd.Flags()
	flags.StringVar(&cfg.Mode, "mode", "", "detection mode: pd or ra")
	flags.IntVar(&cfg.PrefixLen, "prefix-len", -1, "IPv6 prefix length to watch")
	flags.StringVarP(&cfg.Interface, "interface", "i", "", "network interface to watch")
	flags.StringVar(&cfg.Target, "target", "networkd-prefix-changed.target", "systemd target to restart")
	flags.StringVar(&cfg.StateFile, "state-file", "/run/networkd-prefix-watcher/state.json", "state file path")
	flags.StringVar(&cfg.EnvFile, "env-file", "/run/networkd-prefix-watcher/prefix.env", "environment file path")
	flags.DurationVar(&cfg.Debounce, "debounce", 2*time.Second, "debounce duration for netlink events")
	flags.BoolVar(&cfg.Once, "once", false, "check once and exit")
	flags.BoolVar(&cfg.TriggerOnStart, "trigger-on-start", false, "trigger when no previous state exists")

	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("parse flags: %w", err)
	})

	return cmd
}
