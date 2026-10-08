package engine

import "fmt"

// AdvanceWeaponObserved applies one isolated pass with passive point and
// rectangle observers. Queries are reported before native damage/absorption;
// the callbacks are cleared on every return and are never copied by Load.
func (f *WorldForecast) AdvanceWeaponObserved(input Input, point func(WeaponPointImpact), rectangle func(WeaponRectImpact)) (ForecastResult, error) {
	if f.world == nil {
		return ForecastResult{}, fmt.Errorf("forecast has not been loaded")
	}
	if f.world.Weapons == nil {
		return f.result(), fmt.Errorf("weapon observation requires a weapon runtime")
	}
	weapons := f.world.Weapons
	weapons.context.ObservePointImpact, weapons.context.ObserveRectImpact = point, rectangle
	defer func() {
		weapons.context.ObservePointImpact, weapons.context.ObserveRectImpact = nil, nil
	}()
	return f.Advance(input)
}
