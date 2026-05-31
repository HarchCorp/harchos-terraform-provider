package sovereignty

import (
        "context"
        "fmt"
        "strings"

        "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
        "github.com/hashicorp/terraform-plugin-framework/schema/validator"
        "github.com/hashicorp/terraform-plugin-framework/types"
)

// Sovereignty levels ordered from most restrictive to least restrictive.
// Sovereignty cannot be downgraded: strict > flexible > regional > global.
const (
        LevelStrict   = "strict"
        LevelFlexible  = "flexible"
        LevelRegional  = "regional"
        LevelGlobal    = "global"
)

// sovereigntyRank maps sovereignty levels to their enforcement rank.
// Higher rank means more restrictive. Downgrade prevention means
// a resource cannot move from a higher rank to a lower one.
var sovereigntyRank = map[string]int{
        LevelStrict:   4,
        LevelFlexible:  3,
        LevelRegional:  2,
        LevelGlobal:    1,
}

// ValidLevels returns all valid sovereignty level strings.
func ValidLevels() []string {
        return []string{LevelStrict, LevelFlexible, LevelRegional, LevelGlobal}
}

// IsValid checks if a sovereignty level is recognized.
func IsValid(level string) bool {
        _, ok := sovereigntyRank[strings.ToLower(level)]
        return ok
}

// CanTransition determines if a sovereignty transition is allowed.
// Transitions are only allowed if the new level is equal to or more
// restrictive than the current level. Downgrades are prohibited.
func CanTransition(current, proposed string) error {
        currentRank, ok := sovereigntyRank[strings.ToLower(current)]
        if !ok {
                return fmt.Errorf("unknown current sovereignty level: %q", current)
        }

        proposedRank, ok := sovereigntyRank[strings.ToLower(proposed)]
        if !ok {
                return fmt.Errorf("unknown proposed sovereignty level: %q", proposed)
        }

        if proposedRank < currentRank {
                return &DowngradeError{
                        Current:   current,
                        Proposed:  proposed,
                        CurrentRank: currentRank,
                        ProposedRank: proposedRank,
                }
        }

        return nil
}

// DowngradeError is returned when a sovereignty downgrade is attempted.
type DowngradeError struct {
        Current     string
        Proposed    string
        CurrentRank int
        ProposedRank int
}

func (e *DowngradeError) Error() string {
        return fmt.Sprintf(
                "sovereignty cannot be downgraded from %q (rank %d) to %q (rank %d); "+
                        "HarchOS enforces strict > flexible > regional > global escalation policy",
                e.Current, e.CurrentRank, e.Proposed, e.ProposedRank,
        )
}

// SovereigntyValidator validates that a sovereignty level is valid
// and optionally that transitions respect the escalation policy.
type SovereigntyValidator struct {
        // PreviousLevel, if set, enables downgrade prevention checks.
        PreviousLevel types.String
}

var _ validator.String = SovereigntyValidator{}

// Description returns a human-readable description of the validator.
func (v SovereigntyValidator) Description(_ context.Context) string {
        return "validates that sovereignty is one of: strict, flexible, regional, global; and cannot be downgraded"
}

// MarkdownDescription returns a markdown-formatted description.
func (v SovereigntyValidator) MarkdownDescription(_ context.Context) string {
        return "Validates that sovereignty is one of: `strict`, `flexible`, `regional`, `global`. " +
                "Enforces escalation policy: sovereignty cannot be downgraded (strict > flexible > regional > global)."
}

// ValidateString performs the sovereignty validation.
func (v SovereigntyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
        if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
                return
        }

        value := req.ConfigValue.ValueString()

        // Check that the value is a valid sovereignty level
        if !IsValid(value) {
                resp.Diagnostics.AddAttributeError(
                        req.Path,
                        "Invalid sovereignty level",
                        fmt.Sprintf("Expected one of %v, got %q", ValidLevels(), value),
                )
                return
        }

        // Check downgrade prevention if previous level is known
        if !v.PreviousLevel.IsNull() && !v.PreviousLevel.IsUnknown() {
                previous := v.PreviousLevel.ValueString()
                if err := CanTransition(previous, value); err != nil {
                        resp.Diagnostics.AddAttributeError(
                                req.Path,
                                "Sovereignty downgrade not allowed",
                                err.Error(),
                        )
                }
        }
}

// SovereigntyLevelValidator returns a validator that checks sovereignty values
// are valid (strict, flexible, regional, global).
func SovereigntyLevelValidator() validator.String {
        return stringvalidator.OneOfCaseInsensitive(ValidLevels()...)
}

// NewSovereigntyValidator creates a new SovereigntyValidator with optional
// downgrade prevention based on a previous sovereignty level.
func NewSovereigntyValidator(previousLevel types.String) SovereigntyValidator {
        return SovereigntyValidator{
                PreviousLevel: previousLevel,
        }
}

// EffectiveSovereignty determines the effective sovereignty level for a
// resource by comparing the provider-level and resource-level settings.
// The more restrictive level always takes precedence.
func EffectiveSovereignty(providerLevel, resourceLevel string) (string, error) {
        if providerLevel == "" && resourceLevel == "" {
                return "", fmt.Errorf("sovereignty must be set at either provider or resource level")
        }

        if providerLevel == "" {
                if !IsValid(resourceLevel) {
                        return "", fmt.Errorf("invalid resource-level sovereignty: %q", resourceLevel)
                }
                return resourceLevel, nil
        }

        if resourceLevel == "" {
                if !IsValid(providerLevel) {
                        return "", fmt.Errorf("invalid provider-level sovereignty: %q", providerLevel)
                }
                return providerLevel, nil
        }

        providerRank, ok := sovereigntyRank[strings.ToLower(providerLevel)]
        if !ok {
                return "", fmt.Errorf("invalid provider-level sovereignty: %q", providerLevel)
        }

        resourceRank, ok := sovereigntyRank[strings.ToLower(resourceLevel)]
        if !ok {
                return "", fmt.Errorf("invalid resource-level sovereignty: %q", resourceLevel)
        }

        // Return the more restrictive level
        if providerRank >= resourceRank {
                return strings.ToLower(providerLevel), nil
        }
        return strings.ToLower(resourceLevel), nil
}

// ValidateSovereigntyTransition is a helper that validates a sovereignty
// change during resource update. It returns diagnostics if the transition
// is invalid.
func ValidateSovereigntyTransition(currentState, plannedState types.String) error {
        if currentState.IsNull() || plannedState.IsNull() {
                return nil
        }
        if currentState.IsUnknown() || plannedState.IsUnknown() {
                return nil
        }

        current := currentState.ValueString()
        planned := plannedState.ValueString()

        if current == planned {
                return nil
        }

        return CanTransition(current, planned)
}
