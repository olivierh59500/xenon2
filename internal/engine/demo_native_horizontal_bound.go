package engine

const (
	nativeHorizontalMinimumX  = 14
	nativeHorizontalMaximumX  = 304
	nativeHorizontalInertias  = 13
	nativeHorizontalStates    = (nativeHorizontalMaximumX - nativeHorizontalMinimumX + 1) * nativeHorizontalInertias
	nativeHorizontalEdges     = nativeHorizontalStates * 3
	nativeHorizontalUnreached = ^uint16(0)
)

// nativeHorizontalBound caches the exact horizontal travel time with terrain
// and vertical motion omitted. Each speed's graph and working arrays live behind
// a pointer so copying a planner does not copy thousands of states.
// A cache belongs to one planner and must not be used concurrently.
type nativeHorizontalBound struct {
	speeds [3]*nativeHorizontalTable
}

type nativeHorizontalTable struct {
	starts       [nativeHorizontalStates + 1]uint16
	predecessors [nativeHorizontalEdges]uint16
	distances    [nativeHorizontalStates]uint16
	queue        [nativeHorizontalStates]uint16
	targetX      int
}

func nativeHorizontalIndex(x, inertia int) int {
	return (x-nativeHorizontalMinimumX)*nativeHorizontalInertias + inertia + 6
}

func nativeHorizontalPlayer(index, speed int) PlayerMotionState {
	return PlayerMotionState{X: nativeHorizontalMinimumX + index/nativeHorizontalInertias,
		Y: 100, Inertia: index%nativeHorizontalInertias - 6, SpeedTier: speed}
}

var nativeHorizontalInputs = [3]MotionInput{{Left: true}, {}, {Right: true}}

func newNativeHorizontalTable(speed int) *nativeHorizontalTable {
	table := &nativeHorizontalTable{targetX: -1}
	context := MotionContext{BaseScrollStep: 1}
	// Build the reverse graph from the canonical player callback, retaining
	// its signed rounding, immediate held speed and inertia at either clamp.
	for index := range nativeHorizontalStates {
		for _, input := range nativeHorizontalInputs {
			player := nativeHorizontalPlayer(index, speed)
			player.Advance(input, context)
			table.starts[nativeHorizontalIndex(player.X, player.Inertia)+1]++
		}
	}
	for index := 1; index < len(table.starts); index++ {
		table.starts[index] += table.starts[index-1]
	}
	var cursor [nativeHorizontalStates]uint16
	copy(cursor[:], table.starts[:])
	for index := range nativeHorizontalStates {
		for _, input := range nativeHorizontalInputs {
			player := nativeHorizontalPlayer(index, speed)
			player.Advance(input, context)
			next := nativeHorizontalIndex(player.X, player.Inertia)
			table.predecessors[cursor[next]] = uint16(index)
			cursor[next]++
		}
	}
	return table
}

func (table *nativeHorizontalTable) target(x int) {
	for index := range table.distances {
		table.distances[index] = nativeHorizontalUnreached
	}
	count := 0
	// The real route goal permits every final inertia, rather than requiring
	// the ship to stop at the requested horizontal coordinate.
	for inertia := -6; inertia <= 6; inertia++ {
		index := nativeHorizontalIndex(x, inertia)
		table.distances[index] = 0
		table.queue[count] = uint16(index)
		count++
	}
	for at := 0; at < count; at++ {
		index := table.queue[at]
		for _, previous := range table.predecessors[table.starts[index]:table.starts[index+1]] {
			if table.distances[previous] != nativeHorizontalUnreached {
				continue
			}
			table.distances[previous] = table.distances[index] + 1
			table.queue[count] = previous
			count++
		}
	}
	table.targetX = x
}

// distance returns an admissible lower bound for an ordinary source-motion
// route. Speed is explicit because equipment can change before Player.SpeedTier
// is refreshed. Invalid or unreachable states return an unsupported zero bound.
func (bound *nativeHorizontalBound) distance(x, inertia, targetX, speed int) (int, bool) {
	if x < nativeHorizontalMinimumX || x > nativeHorizontalMaximumX ||
		targetX < nativeHorizontalMinimumX || targetX > nativeHorizontalMaximumX ||
		inertia < -6 || inertia > 6 || speed < 0 || speed >= len(bound.speeds) {
		return 0, false
	}
	table := bound.speeds[speed]
	if table == nil {
		table = newNativeHorizontalTable(speed)
		bound.speeds[speed] = table
	}
	if table.targetX != targetX {
		table.target(targetX)
	}
	distance := table.distances[nativeHorizontalIndex(x, inertia)]
	if distance == nativeHorizontalUnreached {
		return 0, false
	}
	return int(distance), true
}
