package game

import "math"

const cellSize float32 = 100

// CellKey identifies a cell in the spatial grid.
type CellKey struct {
	X, Y, Z int32
}

// SpatialGrid accelerates proximity queries for ships and projectiles.
// Rebuilt from scratch once per tick (cheap: one map clear + N inserts).
type SpatialGrid struct {
	cells map[CellKey][]*Ship
}

// NewSpatialGrid creates an empty spatial grid.
func NewSpatialGrid() *SpatialGrid {
	return &SpatialGrid{cells: make(map[CellKey][]*Ship)}
}

func posToCell(pos Vec3) CellKey {
	return CellKey{
		X: int32(math.Floor(float64(pos.X / cellSize))),
		Y: int32(math.Floor(float64(pos.Y / cellSize))),
		Z: int32(math.Floor(float64(pos.Z / cellSize))),
	}
}

// Rebuild clears and repopulates the grid with all living ships.
func (g *SpatialGrid) Rebuild(ships map[string]*Ship) {
	for k := range g.cells {
		delete(g.cells, k)
	}
	for _, ship := range ships {
		if !ship.IsAlive {
			continue
		}
		key := posToCell(ship.Position)
		g.cells[key] = append(g.cells[key], ship)
	}
}

// GetNearby returns all ships within the cells covered by a bounding sphere.
// The caller must perform a precise distance check on the results.
func (g *SpatialGrid) GetNearby(pos Vec3, radius float32) []*Ship {
	minKey := CellKey{
		X: int32(math.Floor(float64((pos.X - radius) / cellSize))),
		Y: int32(math.Floor(float64((pos.Y - radius) / cellSize))),
		Z: int32(math.Floor(float64((pos.Z - radius) / cellSize))),
	}
	maxKey := CellKey{
		X: int32(math.Floor(float64((pos.X + radius) / cellSize))),
		Y: int32(math.Floor(float64((pos.Y + radius) / cellSize))),
		Z: int32(math.Floor(float64((pos.Z + radius) / cellSize))),
	}

	var result []*Ship
	for x := minKey.X; x <= maxKey.X; x++ {
		for y := minKey.Y; y <= maxKey.Y; y++ {
			for z := minKey.Z; z <= maxKey.Z; z++ {
				result = append(result, g.cells[CellKey{x, y, z}]...)
			}
		}
	}
	return result
}
