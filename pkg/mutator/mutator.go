package mutator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// Status represents the outcome of testing a mutation.
type Status int

const (
	StatusPending    Status = iota
	StatusKilled            // A test failed — mutation was detected
	StatusSurvived          // All tests passed — mutation was NOT detected
	StatusBuildError        // Mutated code failed to compile
	StatusTimeout           // Test execution exceeded time limit
	StatusSkipped           // Skipped (e.g., uncovered line, cached)
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusKilled:
		return "killed"
	case StatusSurvived:
		return "survived"
	case StatusBuildError:
		return "build_error"
	case StatusTimeout:
		return "timeout"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// MutationType describes what kind of AST transformation to perform.
type MutationType int

const (
	MutBinaryOp    MutationType = iota // Change a binary operator token
	MutBoolLiteral                     // Swap true/false identifier
	MutNilCheck                        // Empty the body of an if-block
	MutReturnValue                     // Replace a return expression with zero value
	MutAssignOp                        // Change a compound assignment operator token
	MutRemoveElse                      // Remove an else block
	MutRemoveCase                      // Empty a case clause body
)

// Mutation describes a single source code mutation declaratively.
// The runner uses File, Line, Column, Type, Original, and Replacement
// to locate and apply the mutation on a freshly parsed AST.
type Mutation struct {
	File        string       // Absolute path to the source file
	Package     string       // Package import path
	Line        int          // Line number in the original source
	Column      int          // Column number
	Original    string       // Original token/value at mutation site
	Replacement string       // Replacement token/value
	Description string       // e.g., "replaced + with -"
	Type        MutationType // What kind of mutation
}

func (m Mutation) String() string {
	return fmt.Sprintf("%s:%d — %s", m.File, m.Line, m.Description)
}

// Result holds the outcome of testing one mutation.
type Result struct {
	Mutation Mutation `json:"mutation"`
	Status   Status   `json:"status"`
	Duration float64  `json:"duration"` // Seconds
	Output   string   `json:"output"`   // Test output (stderr/stdout) on failure
}

// Mutator generates mutations for a given AST file.
type Mutator interface {
	Name() string
	Mutate(fset *token.FileSet, file *ast.File, info *types.Info) []Mutation
}

// Registry holds all registered mutators.
var registry []Mutator

// Register adds a mutator to the global registry.
func Register(m Mutator) {
	registry = append(registry, m)
}

// All returns all registered mutators.
func All() []Mutator {
	return registry
}
