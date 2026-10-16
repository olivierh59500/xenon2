package engine

import "xenon2/internal/visualassets"

// SecondTerrainCells retains the final arena's intact flags separately from the
// mutable map. Bullets clear one cell; moving turrets can restore that cell.
type SecondTerrainCells struct {
	Cells  []visualassets.GuardianTerrainCell
	Intact []bool
}

func NewSecondTerrainCells(cells []visualassets.GuardianTerrainCell) *SecondTerrainCells {
	state := &SecondTerrainCells{Cells: append([]visualassets.GuardianTerrainCell(nil), cells...), Intact: make([]bool, len(cells))}
	for i, cell := range cells {
		state.Intact[i] = !cell.InitiallyDestroyed
	}
	return state
}
func secondTerrainQuadrant(rect CollisionRect) int {
	q := 0
	if rect.Top <= 128 {
		q++
	}
	if rect.Left <= 128 {
		q += 2
	}
	return q
}
func (s *SecondTerrainCells) FindBullet(rect CollisionRect, scrollY int) int {
	rect.Top += scrollY
	rect.Bottom += scrollY
	if rect.Empty() || rect.Top >= 432 {
		return -1
	}
	quadrant := secondTerrainQuadrant(rect)
	for i, cell := range s.Cells {
		if cell.Quadrant != quadrant || !s.Intact[i] {
			continue
		}
		if cell.X > rect.Right {
			break
		}
		if (CollisionRect{Left: cell.X, Top: cell.WorldY, Right: cell.X + 15, Bottom: cell.WorldY + 15}).Intersects(rect) {
			return i
		}
	}
	return -1
}
func (s *SecondTerrainCells) RestoreOverlap(rect CollisionRect, scrollY int) int {
	rect.Top += scrollY
	rect.Bottom += scrollY
	if rect.Empty() {
		return -1
	}
	quadrant := secondTerrainQuadrant(rect)
	for i, cell := range s.Cells {
		if cell.Quadrant != quadrant || s.Intact[i] {
			continue
		}
		if (CollisionRect{Left: cell.X, Top: cell.WorldY, Right: cell.X + 15, Bottom: cell.WorldY + 15}).Intersects(rect) {
			s.Intact[i] = true
			return i
		}
	}
	return -1
}
