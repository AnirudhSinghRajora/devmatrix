package db

import (
	"context"

	"github.com/rs/zerolog/log"
)

// DBWriter processes coin awards and kill records asynchronously
// so the game loop never blocks on database writes.
type DBWriter struct {
	queries *Queries
	CoinCh  chan CoinAward
	KillCh  chan KillRecord
}

func NewDBWriter(queries *Queries) *DBWriter {
	return &DBWriter{
		queries: queries,
		CoinCh:  make(chan CoinAward, 256),
		KillCh:  make(chan KillRecord, 256),
	}
}

func (w *DBWriter) Run(ctx context.Context) {
	log.Info().Msg("DB writer started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("DB writer stopped")
			return
		case award := <-w.CoinCh:
			if err := w.queries.AwardCoins(ctx, award.PlayerID, award.Amount); err != nil {
				log.Error().Err(err).Str("player", award.PlayerID.String()).Msg("failed to award coins")
			}
		case kill := <-w.KillCh:
			if err := w.queries.RecordKill(ctx, kill.KillerID, kill.VictimID); err != nil {
				log.Error().Err(err).Msg("failed to record kill")
			}
		}
	}
}
