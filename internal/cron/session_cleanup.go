package cron

import (
	"github.com/rs/zerolog/log"
)

// CleanupSessionsJob runs periodically to clean up expired sessions
func CleanupSessionsJob(ctx *JobCtx) {
	log.Debug().Msg("Running session cleanup job")
	ctx.App.Database.CleanupExpiredSessions()
}
