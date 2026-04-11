package llm

import (
	"context"
	"fmt"
	"sort"

	"github.com/DevMatrix/server/internal/game"
	"github.com/rs/zerolog/log"
)

// Service manages the LLM processing pipeline: queue, workers, and client.
type Service struct {
	client   *Client // nil when in mock mode
	workers  int
	reqCh    <-chan game.LLMRequest
	resultCh chan<- game.LLMResult
}

// NewService creates a Service.
// If llmURL is empty, the service runs in mock mode (keyword-based parsing).
func NewService(llmURL string, workers int, reqCh <-chan game.LLMRequest, resultCh chan<- game.LLMResult) *Service {
	var c *Client
	if llmURL != "" {
		c = NewClient(llmURL)
	}
	return &Service{
		client:   c,
		workers:  workers,
		reqCh:    reqCh,
		resultCh: resultCh,
	}
}

// Run starts worker goroutines. Blocks until ctx is cancelled.
func (s *Service) Run(ctx context.Context) {
	mode := "mock"
	if s.client != nil {
		mode = "llm"
	}
	log.Info().Int("workers", s.workers).Str("mode", mode).Msg("LLM service started")

	for i := 0; i < s.workers; i++ {
		go s.worker(ctx, i)
	}

	<-ctx.Done()
	log.Info().Msg("LLM service stopped")
}

func (s *Service) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-s.reqCh:
			if !ok {
				return
			}
			result := s.processRequest(ctx, req)
			select {
			case s.resultCh <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *Service) processRequest(ctx context.Context, req game.LLMRequest) game.LLMResult {
	if s.client == nil {
		return s.processMock(req)
	}
	return s.processLLM(ctx, req)
}

func (s *Service) processMock(req game.LLMRequest) game.LLMResult {
	behavior, err := MockGenerate(req.PromptText)
	if err != nil {
		return game.LLMResult{PlayerID: req.PlayerID, Error: err}
	}
	log.Debug().
		Str("player", req.PlayerID).
		Str("movement", behavior.Primary.Movement).
		Msg("mock behavior generated")
	return game.LLMResult{PlayerID: req.PlayerID, Behavior: behavior}
}

func (s *Service) processLLM(ctx context.Context, req game.LLMRequest) game.LLMResult {
	shipInfo := ShipInfo{
		HealthPct: req.HealthPct,
		ShieldPct: req.ShieldPct,
		Pos:       req.ShipPos,
	}

	systemPrompt := BuildSystemPrompt(req.AITier, shipInfo, nil) // TODO: pass nearby enemies
	userPrompt := fmt.Sprintf("Captain: %q", req.PromptText)

	text, err := s.client.Generate(ctx, systemPrompt, userPrompt)
	if err != nil {
		return game.LLMResult{PlayerID: req.PlayerID, Error: fmt.Errorf("LLM call failed: %w", err)}
	}

	behavior, err := game.ParseBehaviorJSON(text)
	if err != nil {
		log.Warn().
			Str("player", req.PlayerID).
			Str("raw", text).
			Err(err).
			Msg("LLM returned invalid behavior JSON")
		return game.LLMResult{PlayerID: req.PlayerID, Error: fmt.Errorf("AI processor failed to interpret that command: %v", err)}
	}

	log.Info().
		Str("player", req.PlayerID).
		Str("movement", behavior.Primary.Movement).
		Msg("LLM behavior applied")
	return game.LLMResult{PlayerID: req.PlayerID, Behavior: behavior}
}

// BuildNearbyEnemies creates sorted enemy info from the game state ships map.
// Called by the engine before submitting an LLM request.
func BuildNearbyEnemies(ship *game.Ship, ships map[string]*game.Ship) []EnemyInfo {
	enemies := make([]EnemyInfo, 0, len(ships)-1)
	for _, s := range ships {
		if s.ID == ship.ID {
			continue
		}
		enemies = append(enemies, EnemyInfo{
			ID:        s.ID,
			Distance:  ship.Position.DistTo(s.Position),
			HealthPct: s.HealthPct(),
			ShieldPct: s.ShieldPct(),
		})
	}
	sort.Slice(enemies, func(i, j int) bool {
		return enemies[i].Distance < enemies[j].Distance
	})
	return enemies
}
