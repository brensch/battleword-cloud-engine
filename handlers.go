package main

import (
	"context"
	"net/http"
	"time"

	"github.com/brensch/battleword"
	"github.com/gin-gonic/gin"
)

type StartMatchRequest struct {
	Letters int      `json:"letters,omitempty"`
	Games   int      `json:"games,omitempty"`
	Players []string `json:"players,omitempty"`
}

type StartMatchResponse struct {
	UUID    string                        `json:"uuid,omitempty"`
	Players []battleword.PlayerDefinition `json:"players,omitempty"`
}

func (s *apiStore) handleStartMatch(c *gin.Context) {

	var req StartMatchRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	match, err := battleword.InitMatch(s.log, battleword.AllWords, battleword.CommonWords, req.Players, req.Letters, 6, req.Games)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	snap := match.Snapshot()
	_, err = s.fsClient.Collection(FirestoreMatchCollection).Doc(snap.UUID).Set(context.Background(), snap)
	if err != nil {
		s.log.WithError(err).Error("failed to write match to firestore")
	}

	// background match calls here.
	go func() {
		updateTicker := time.NewTicker(1 * time.Second)
		finishedCHAN := make(chan struct{})
		go func() {
			finished := false
			for {
				select {
				case <-updateTicker.C:
				case <-finishedCHAN:
					finished = true
				}
				matchSnap := match.Snapshot()
				_, err = s.fsClient.Collection(FirestoreMatchCollection).Doc(matchSnap.UUID).Set(context.Background(), matchSnap)
				if err != nil {
					s.log.WithError(err).Error("failed to write match to firestore")
				}

				if finished {
					return
				}
			}

		}()

		match.Start()
		finishedCHAN <- struct{}{}
		match.Broadcast()

	}()

	finalSnap := match.Snapshot()
	var playerDefinitions []battleword.PlayerDefinition
	for _, player := range finalSnap.Players {
		playerDefinitions = append(playerDefinitions, player.Definition)
	}

	c.JSON(200, StartMatchResponse{
		UUID:    finalSnap.UUID,
		Players: playerDefinitions,
	})
}
