package cmd

import (
	"fmt"

	"github.com/Emyrk/screeps-watcher/internal/version"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/coder/serpent"
)

var (
	GroupLogs = &serpent.Group{
		Parent:      nil,
		Name:        "Logs",
		YAML:        "log",
		Description: "Logging options.",
	}
)

type Root struct {
	LogHuman bool
	LogLevel string
}

func New() *Root {
	return &Root{}
}

func (r *Root) RootCmd() *serpent.Command {
	cmd := &serpent.Command{
		Use: "screeps-watcher",
		Options: serpent.OptionSet{
			{
				Name:        "log-human",
				Description: "Output human friendly logs instead of json.",
				Flag:        "log-human",
				Env:         "SCREEPS_LOG_HUMAN",
				YAML:        "log_human",
				Default:     "false",
				Value:       serpent.BoolOf(&r.LogHuman),
				Group:       GroupLogs,
			},
			{
				Name:        "log-level",
				Description: "Only this level and above is logged.",
				Flag:        "log-level",
				Env:         "SCREEPS_LOG_LEVEL",
				YAML:        "log_level",
				Default:     "debug",
				Value:       serpent.EnumOf(&r.LogLevel, "trace", "debug", "info", "warn", "error", "fatal", "panic"),
				Group:       GroupLogs,
			},
		},
	}

	cmd.AddSubcommands(
		versionCmd(),
		r.WatchCmd(),
		r.segment(),
	)

	return cmd
}

func (r *Root) Logger(inv *serpent.Invocation) zerolog.Logger {
	out := inv.Stderr
	if r.LogHuman {
		// human format it!
		out = zerolog.ConsoleWriter{Out: inv.Stderr}
	}

	var logger zerolog.Logger
	logger = zerolog.New(out).With().Timestamp().Logger()
	// This helps us identify when a log line is from a different boot.
	logger = logger.With().Str("boot_id", uuid.NewString()[:4]).Logger()
	lvl, err := zerolog.ParseLevel(r.LogLevel)
	if err != nil {
		logger.Error().Err(err).Str("level", r.LogLevel).Msg("failed to parse log level")
		lvl = zerolog.InfoLevel
	}
	logger = logger.Level(lvl)
	return logger
}

func versionCmd() *serpent.Command {
	return &serpent.Command{
		Use:   "version",
		Short: "Print the version information",
		Handler: func(inv *serpent.Invocation) error {
			_, _ = fmt.Printf("Git Tag: %s\n", version.GitTag)
			_, _ = fmt.Printf("Git Commit: %s\n", version.GitCommit)
			_, _ = fmt.Printf("Build Time: %s\n", version.BuildTime)
			return nil
		},
	}
}
