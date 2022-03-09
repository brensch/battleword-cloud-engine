package main

import (
	"context"
	"fmt"

	"github.com/brensch/battleword"
)

type SavedSolver struct {
	URI        string                      `json:"uri,omitempty"`
	Definition battleword.PlayerDefinition `json:"definition,omitempty"`
}

func (s *store) OnboardURI(ctx context.Context, uri string) (battleword.PlayerDefinition, string, error) {

	definition, err := battleword.GetDefinition(uri)
	if err != nil {
		return battleword.PlayerDefinition{}, "", err
	}

	docs, err := s.fsClient.Collection(FirestoreSolverCollection).
		Where("Definition.Name", "==", definition.Name).
		Documents(ctx).
		GetAll()
	if err != nil {
		return battleword.PlayerDefinition{}, "", err
	}

	if len(docs) > 0 {
		return battleword.PlayerDefinition{}, "", fmt.Errorf("someone's already called %s", definition.Name)
	}

	save := SavedSolver{
		URI:        uri,
		Definition: definition,
	}

	ref, _, err := s.fsClient.Collection(FirestoreSolverCollection).Add(context.Background(), save)
	if err != nil {
		return battleword.PlayerDefinition{}, "", err
	}

	return definition, ref.ID, nil

}
