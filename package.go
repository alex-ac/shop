package shop

import (
	"fmt"
	"time"
)

func IsValidPackageName(name string) bool {
	// The case when state machine is simplier than regex:
	// (
	//   [A-Za-z]
	//   (
	//     [A-Za-z0-9._-]*
	//     [A-Za-z0-9]
	//   )?
	//   /
	// )*
	// [A-Za-z]
	// (
	//   [A-Za-z0-9._-]*
	//   [A-Za-z0-9]
	// )?

	// stateFunc returns three values:
	// - ok - is argument a valid symbol in this state.
	// - final - is end of input valid after this symbol.
	// - next - next state
	type stateFunc func(rune) (ok, final bool, next stateFunc)
	var state0, state1, state2 stateFunc

	state0, state1, state2 = func(r rune) (bool, bool, stateFunc) {
		// state0:
		// we're either in the begining of the string or just after '/'.
		// We only accept [A-Za-z] here.
		// It's always ok to finish input after this state.
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		next := state1
		return ok, true, next
	}, func(r rune) (bool, bool, stateFunc) {
		// state1:
		// We're anywhere in the input, but not after '.', '_', '-' or '/'
		// We accept [A-Za-z0-9._/-! here.
		// It's ok to finish input if r is in [A-Za-z0-9]
		// We go back to state0 if r is '/' here.
		// We go to state2 if r is '.', '_', or '-' here.
		isPunct := r == '.' || r == '_' || r == '-'
		isSlash := r == '/'
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || isPunct || isSlash
		next := state1
		final := true
		switch {
		case isPunct:
			next = state2
			final = false
		case isSlash:
			next = state0
			final = false
		}
		return ok, final, next
	}, func(r rune) (bool, bool, stateFunc) {
		// state2:
		// We're just after '.', '_', or '-' here.
		// We only accept [A-Za-z0-9] here.
		// It's always ok to finish input after this state.
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		next := state1
		return ok, true, next
	}

	// We start in state0, and it's not ok to finish before first symbol.
	state, final := state0, false
	for _, r := range name {
		var ok bool
		ok, final, state = state(r)
		// If state have not accepted r, then validation is failed.
		if !ok {
			return false
		}
	}

	// We're at the end of input. It's only valid if it's ok to finish now.
	return final
}

type Package struct {
	ApiVersion  string        `json:"api_version"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Repo        string        `json:"repo,omitempty"`
	UpdatedAt   UnixTimestamp `json:"package"`
}

func NewPackage(name, description, repo string) (pkg Package, err error) {
	if !IsValidPackageName(name) {
		err = fmt.Errorf("%w: %s", ErrInvalidPackageName, name)
	} else {
		pkg = Package{
			ApiVersion:  LatestVersion,
			Name:        name,
			Description: description,
			Repo:        repo,
			UpdatedAt:   UnixTimestamp{time.Now()},
		}
	}
	return
}
