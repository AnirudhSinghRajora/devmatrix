package game

import "math"

// hullShapes maps hull IDs to their compound collision shapes.
// Offsets are in local space (ship-forward = +Z).
var hullShapes = map[string][]CollisionSphere{
	"hull_basic": { // Striker — small, elongated
		{Offset: Vec3{}, Radius: 1.2},
		{Offset: Vec3{Z: 1.5}, Radius: 0.8},
	},
	"hull_medium": { // Challenger — wide wings
		{Offset: Vec3{}, Radius: 1.5},
		{Offset: Vec3{X: -1.8}, Radius: 0.8},
		{Offset: Vec3{X: 1.8}, Radius: 0.8},
	},
	"hull_heavy": { // Imperial — long and bulky
		{Offset: Vec3{}, Radius: 2.0},
		{Offset: Vec3{Z: 2.0}, Radius: 1.5},
		{Offset: Vec3{Z: -2.0}, Radius: 1.2},
	},
	"hull_stealth": { // Omen — slim with tail
		{Offset: Vec3{}, Radius: 1.0},
		{Offset: Vec3{Z: -1.5}, Radius: 0.6},
	},
}

// defaultShape is used when the hull ID has no compound shape defined.
var defaultShape = []CollisionSphere{
	{Offset: Vec3{}, Radius: 2.0},
}

// hullShapeFor returns the compound collision shape for a hull ID.
func hullShapeFor(hullID string) []CollisionSphere {
	if s, ok := hullShapes[hullID]; ok {
		return s
	}
	return defaultShape
}

// boundingRadius returns the smallest sphere centred at the ship origin that
// encloses the entire compound shape (used for broad-phase).
func boundingRadius(shape []CollisionSphere) float32 {
	var max float32
	for _, s := range shape {
		r := s.Offset.Length() + s.Radius
		if r > max {
			max = r
		}
	}
	return max
}

// worldSubSphere returns the world-space position of a sub-sphere given the
// ship's position and rotation.
func worldSubSphere(shipPos Vec3, shipRot Quaternion, s CollisionSphere) (pos Vec3, radius float32) {
	return shipPos.Add(shipRot.Rotate(s.Offset)), s.Radius
}

// compoundOverlap tests all sub-sphere pairs between two ships and returns the
// contact with deepest penetration.  Returns ok=false if no overlap.
func compoundOverlap(
	posA Vec3, rotA Quaternion, shapeA []CollisionSphere,
	posB Vec3, rotB Quaternion, shapeB []CollisionSphere,
) (normal Vec3, penetration float32, ok bool) {
	var bestPen float32 = -1
	var bestNormal Vec3

	for _, sa := range shapeA {
		wa, ra := worldSubSphere(posA, rotA, sa)
		for _, sb := range shapeB {
			wb, rb := worldSubSphere(posB, rotB, sb)
			diff := wa.Sub(wb)
			distSq := diff.Dot(diff)
			radSum := ra + rb
			if distSq >= radSum*radSum {
				continue
			}
			dist := float32(math.Sqrt(float64(distSq)))
			pen := radSum - dist
			if pen > bestPen {
				bestPen = pen
				if dist < 1e-6 {
					bestNormal = Vec3{X: 1}
				} else {
					bestNormal = diff.Scale(1 / dist)
				}
			}
		}
	}

	if bestPen < 0 {
		return Vec3{}, 0, false
	}
	return bestNormal, bestPen, true
}

// compoundRayHit tests if a ray hits any sub-sphere of a ship's compound shape.
func compoundRayHit(origin, dir, shipPos Vec3, shipRot Quaternion, shape []CollisionSphere) bool {
	for _, s := range shape {
		wp, r := worldSubSphere(shipPos, shipRot, s)
		if rayHitsSphere(origin, dir, wp, r) {
			return true
		}
	}
	return false
}

// compoundSphereHit tests if a sphere (e.g. projectile) overlaps any
// sub-sphere of a ship's compound shape.
func compoundSphereHit(pos Vec3, radius float32, shipPos Vec3, shipRot Quaternion, shape []CollisionSphere) bool {
	for _, s := range shape {
		wp, r := worldSubSphere(shipPos, shipRot, s)
		if spheresOverlap(pos, radius, wp, r) {
			return true
		}
	}
	return false
}
