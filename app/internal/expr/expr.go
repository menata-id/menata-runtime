package expr

import (
	"sync"
	"time"

	"github.com/google/cel-go/cel"
)

// Vars is everything an expression may read -- see doc.go's own
// "What an expression can read" section.
type Vars struct {
	Record      map[string]any
	Old         map[string]any
	CurrentUser string
}

// env is the one CEL environment every expression compiles against --
// exactly four variables, zero custom functions (see doc.go's own
// "Guardrail: no I/O, ever"). Built once at package init, not per call.
var env = mustBuildEnv()

func mustBuildEnv() *cel.Env {
	e, err := cel.NewEnv(
		cel.Variable("record", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("old", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("current_user", cel.StringType),
		cel.Variable("today", cel.StringType),
		cel.Variable("now", cel.StringType),
	)
	if err != nil {
		// Only reachable if the four Variable() declarations above are
		// themselves malformed -- a programming error in this package, not
		// something a metadata author's expression text could ever trigger.
		panic("internal/expr: build CEL environment: " + err.Error())
	}
	return e
}

// programCache holds one compiled cel.Program per unique expression STRING
// -- see doc.go's own "Compiled once, not once per evaluation" section for
// why this needs no invalidation.
var programCache sync.Map // string -> cel.Program

func programFor(expression string) (cel.Program, error) {
	if cached, ok := programCache.Load(expression); ok {
		return cached.(cel.Program), nil
	}
	ast, iss := env.Compile(expression)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}
	// A second, concurrent caller compiling the same expression for the
	// first time just overwrites its own equally-valid *cel.Program here --
	// harmless, cheaper than a mutex around a call that's fast anyway.
	programCache.Store(expression, prg)
	return prg, nil
}

// Compile validates expression's syntax (and that it only references the
// four declared variables) without evaluating it -- internal/metadata's own
// load-time validation calls this so a broken expression fails LoadAll
// explicitly, not a random later request.
func Compile(expression string) error {
	_, err := programFor(expression)
	return err
}

// activation builds the map Program.Eval reads variables from. Record/Old
// default to an empty (never nil) map -- see doc.go's own note on why an
// absent "before" state and an absent Record are treated identically to a
// present-but-empty one, not a special nil case an expression author has to
// guard against separately.
func activation(vars Vars) map[string]any {
	record := vars.Record
	if record == nil {
		record = map[string]any{}
	}
	old := vars.Old
	if old == nil {
		old = map[string]any{}
	}
	now := time.Now()
	return map[string]any{
		"record":       record,
		"old":          old,
		"current_user": vars.CurrentUser,
		"today":        now.Format("2006-01-02"),
		"now":          now.Format(time.RFC3339),
	}
}

// Eval evaluates expression against vars, returning its result as a native
// Go value (bool/float64/string/...). Any compile or runtime error
// (including a reference to a field neither record nor old actually has --
// CEL's own map-key-miss error) is returned, never panics.
func Eval(expression string, vars Vars) (any, error) {
	prg, err := programFor(expression)
	if err != nil {
		return nil, err
	}
	out, _, err := prg.Eval(activation(vars))
	if err != nil {
		return nil, err
	}
	return out.Value(), nil
}

// Bool is Eval's boolean-condition convenience -- see doc.go's own
// "Fail-closed, not panic-closed" section: any error, or a result that
// isn't itself a bool, degrades to false rather than propagating.
func Bool(expression string, vars Vars) bool {
	v, err := Eval(expression, vars)
	if err != nil {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
