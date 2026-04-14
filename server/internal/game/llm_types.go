package game

// EnemySnapshot is a lightweight enemy descriptor passed to the LLM pipeline.
type EnemySnapshot struct {
	Username  string
	Distance  float32
	HealthPct float32
	ShieldPct float32
}

// LLMRequest is sent from the engine to the LLM worker pool.
type LLMRequest struct {
	PlayerID   string
	PromptText string
	ShipPos    [3]float32
	HealthPct  float32
	ShieldPct  float32
	AITier     int
	Enemies    []EnemySnapshot
}

// LLMResult is sent from the LLM worker pool back to the engine.
type LLMResult struct {
	PlayerID string
	Behavior *BehaviorSet
	Error    error
}
