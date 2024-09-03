package math

import "golang.org/x/exp/constraints"

func CastTo[T, F constraints.Integer](from F) T {
	return T(from)
}
