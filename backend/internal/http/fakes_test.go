package http

import (
	"context"
	"errors"

	"teamload/internal/ai"
)

// failingProvider always errors, like an unreachable LLM.
type failingProvider struct{}

var errProviderDown = errors.New("provider down")

func (failingProvider) GenerateRolePlan(context.Context, ai.RolePlanInput) (ai.RolePlanOutput, error) {
	return ai.RolePlanOutput{}, errProviderDown
}

func (failingProvider) MatchRoles(context.Context, ai.MatchInput) (ai.MatchOutput, error) {
	return ai.MatchOutput{}, errProviderDown
}

func (failingProvider) DraftRACI(context.Context, ai.RACIInput) (ai.RACIOutput, error) {
	return ai.RACIOutput{}, errProviderDown
}
