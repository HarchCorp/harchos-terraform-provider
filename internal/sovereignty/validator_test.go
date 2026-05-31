package sovereignty

import (
        "context"
        "testing"

        "github.com/hashicorp/terraform-plugin-framework/schema/validator"
        "github.com/hashicorp/terraform-plugin-framework/types"
)

// --- EffectiveSovereignty tests ---

func TestEffectiveSovereignty_ProviderMoreRestrictive(t *testing.T) {
        result, err := EffectiveSovereignty("strict", "global")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result != LevelStrict {
                t.Fatalf("expected %q, got %q", LevelStrict, result)
        }
}

func TestEffectiveSovereignty_ResourceMoreRestrictive(t *testing.T) {
        result, err := EffectiveSovereignty("global", "strict")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result != LevelStrict {
                t.Fatalf("expected %q, got %q", LevelStrict, result)
        }
}

func TestEffectiveSovereignty_SameLevel(t *testing.T) {
        result, err := EffectiveSovereignty("regional", "regional")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result != LevelRegional {
                t.Fatalf("expected %q, got %q", LevelRegional, result)
        }
}

func TestEffectiveSovereignty_ResourceOnly(t *testing.T) {
        result, err := EffectiveSovereignty("", "strict")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result != LevelStrict {
                t.Fatalf("expected %q, got %q", LevelStrict, result)
        }
}

func TestEffectiveSovereignty_ProviderOnly(t *testing.T) {
        result, err := EffectiveSovereignty("regional", "")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if result != LevelRegional {
                t.Fatalf("expected %q, got %q", LevelRegional, result)
        }
}

func TestEffectiveSovereignty_BothEmpty(t *testing.T) {
        _, err := EffectiveSovereignty("", "")
        if err == nil {
                t.Fatal("expected error when both levels are empty")
        }
}

func TestEffectiveSovereignty_InvalidProviderLevel(t *testing.T) {
        _, err := EffectiveSovereignty("invalid", "strict")
        if err == nil {
                t.Fatal("expected error for invalid provider level")
        }
}

func TestEffectiveSovereignty_InvalidResourceLevel(t *testing.T) {
        _, err := EffectiveSovereignty("strict", "invalid")
        if err == nil {
                t.Fatal("expected error for invalid resource level")
        }
}

// --- CanTransition tests ---

func TestCanTransition_ValidTransitions(t *testing.T) {
        tests := []struct {
                current  string
                proposed string
        }{
                {"global", "global"},
                {"global", "regional"},
                {"global", "strict"},
                {"global", "flexible"},
                {"regional", "regional"},
                {"regional", "strict"},
                {"regional", "flexible"},
                {"flexible", "flexible"},
                {"flexible", "strict"},
                {"strict", "strict"},
        }

        for _, tc := range tests {
                t.Run(tc.current+"->"+tc.proposed, func(t *testing.T) {
                        err := CanTransition(tc.current, tc.proposed)
                        if err != nil {
                                t.Fatalf("expected transition %s -> %s to be allowed, got error: %v", tc.current, tc.proposed, err)
                        }
                })
        }
}

func TestCanTransition_InvalidTransitions(t *testing.T) {
        tests := []struct {
                current  string
                proposed string
        }{
                {"strict", "regional"},
                {"strict", "global"},
                {"strict", "flexible"},
                {"regional", "global"},
                {"flexible", "regional"},
                {"flexible", "global"},
        }

        for _, tc := range tests {
                t.Run(tc.current+"->"+tc.proposed, func(t *testing.T) {
                        err := CanTransition(tc.current, tc.proposed)
                        if err == nil {
                                t.Fatalf("expected transition %s -> %s to be rejected", tc.current, tc.proposed)
                        }
                })
        }
}

func TestCanTransition_UnknownCurrent(t *testing.T) {
        err := CanTransition("unknown", "global")
        if err == nil {
                t.Fatal("expected error for unknown current level")
        }
}

func TestCanTransition_UnknownProposed(t *testing.T) {
        err := CanTransition("global", "unknown")
        if err == nil {
                t.Fatal("expected error for unknown proposed level")
        }
}

// --- DowngradeError tests ---

func TestDowngradeError_Format(t *testing.T) {
        err := &DowngradeError{
                Current:      "strict",
                Proposed:     "global",
                CurrentRank:  3,
                ProposedRank: 1,
        }
        msg := err.Error()
        if msg == "" {
                t.Fatal("DowngradeError.Error() should not return empty string")
        }
}

// --- IsValid tests ---

func TestIsValid(t *testing.T) {
        tests := []struct {
                level string
                valid bool
        }{
                {"strict", true},
                {"flexible", true},
                {"regional", true},
                {"global", true},
                {"STRICT", true},
                {"Regional", true},
                {"Flexible", true},
                {"invalid", false},
                {"", false},
        }

        for _, tc := range tests {
                t.Run(tc.level, func(t *testing.T) {
                        if IsValid(tc.level) != tc.valid {
                                t.Fatalf("IsValid(%q) = %v, want %v", tc.level, !tc.valid, tc.valid)
                        }
                })
        }
}

// --- ValidateSovereigntyTransition tests ---

func TestValidateSovereigntyTransition_Valid(t *testing.T) {
        err := ValidateSovereigntyTransition(
                types.StringValue("global"),
                types.StringValue("strict"),
        )
        if err != nil {
                t.Fatalf("expected valid transition, got error: %v", err)
        }
}

func TestValidateSovereigntyTransition_Invalid(t *testing.T) {
        err := ValidateSovereigntyTransition(
                types.StringValue("strict"),
                types.StringValue("global"),
        )
        if err == nil {
                t.Fatal("expected error for invalid transition")
        }
}

func TestValidateSovereigntyTransition_SameLevel(t *testing.T) {
        err := ValidateSovereigntyTransition(
                types.StringValue("regional"),
                types.StringValue("regional"),
        )
        if err != nil {
                t.Fatalf("expected no error for same level, got: %v", err)
        }
}

func TestValidateSovereigntyTransition_NullState(t *testing.T) {
        err := ValidateSovereigntyTransition(
                types.StringNull(),
                types.StringValue("strict"),
        )
        if err != nil {
                t.Fatalf("expected no error for null current, got: %v", err)
        }
}

func TestValidateSovereigntyTransition_UnknownState(t *testing.T) {
        err := ValidateSovereigntyTransition(
                types.StringUnknown(),
                types.StringValue("strict"),
        )
        if err != nil {
                t.Fatalf("expected no error for unknown current, got: %v", err)
        }
}

// --- SovereigntyValidator.ValidateString tests ---

func TestSovereigntyValidator_ValidateString_Valid(t *testing.T) {
        v := SovereigntyValidator{}

        req := validator.StringRequest{
                ConfigValue: types.StringValue("strict"),
        }
        resp := &validator.StringResponse{}

        v.ValidateString(context.Background(), req, resp)

        if resp.Diagnostics.HasError() {
                t.Fatalf("expected no errors, got: %v", resp.Diagnostics)
        }
}

func TestSovereigntyValidator_ValidateString_Invalid(t *testing.T) {
        v := SovereigntyValidator{}

        req := validator.StringRequest{
                ConfigValue: types.StringValue("invalid"),
        }
        resp := &validator.StringResponse{}

        v.ValidateString(context.Background(), req, resp)

        if !resp.Diagnostics.HasError() {
                t.Fatal("expected errors for invalid sovereignty level")
        }
}

func TestSovereigntyValidator_ValidateString_Null(t *testing.T) {
        v := SovereigntyValidator{}

        req := validator.StringRequest{
                ConfigValue: types.StringNull(),
        }
        resp := &validator.StringResponse{}

        v.ValidateString(context.Background(), req, resp)

        if resp.Diagnostics.HasError() {
                t.Fatalf("expected no errors for null value, got: %v", resp.Diagnostics)
        }
}

func TestSovereigntyValidator_ValidateString_Unknown(t *testing.T) {
        v := SovereigntyValidator{}

        req := validator.StringRequest{
                ConfigValue: types.StringUnknown(),
        }
        resp := &validator.StringResponse{}

        v.ValidateString(context.Background(), req, resp)

        if resp.Diagnostics.HasError() {
                t.Fatalf("expected no errors for unknown value, got: %v", resp.Diagnostics)
        }
}

func TestSovereigntyValidator_ValidateString_DowngradePrevention(t *testing.T) {
        v := SovereigntyValidator{
                PreviousLevel: types.StringValue("strict"),
        }

        req := validator.StringRequest{
                ConfigValue: types.StringValue("global"),
        }
        resp := &validator.StringResponse{}

        v.ValidateString(context.Background(), req, resp)

        if !resp.Diagnostics.HasError() {
                t.Fatal("expected errors for sovereignty downgrade")
        }
}

func TestSovereigntyValidator_ValidateString_UpgradeAllowed(t *testing.T) {
        v := SovereigntyValidator{
                PreviousLevel: types.StringValue("global"),
        }

        req := validator.StringRequest{
                ConfigValue: types.StringValue("strict"),
        }
        resp := &validator.StringResponse{}

        v.ValidateString(context.Background(), req, resp)

        if resp.Diagnostics.HasError() {
                t.Fatalf("expected no errors for sovereignty upgrade, got: %v", resp.Diagnostics)
        }
}

// --- ValidLevels tests ---

func TestValidLevels(t *testing.T) {
        levels := ValidLevels()
        if len(levels) != 4 {
                t.Fatalf("expected 4 valid levels, got %d", len(levels))
        }
        found := make(map[string]bool)
        for _, l := range levels {
                found[l] = true
        }
        if !found[LevelStrict] || !found[LevelFlexible] || !found[LevelRegional] || !found[LevelGlobal] {
                t.Fatal("ValidLevels() missing expected levels")
        }
}
