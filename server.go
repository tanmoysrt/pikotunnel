package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type AccessRuleRequest struct {
	PeerAID string `json:"peer_a_id"`
	PeerBID string `json:"peer_b_id"`
}

func startServer() {
	e := echo.New()
	e.HideBanner = true

	// Auth middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token != config.APIToken {
				return c.JSON(http.StatusUnauthorized, "Unauthorized")
			}
			return next(c)
		}
	})

	// Register routes
	e.POST("/peers", createPeer)
	e.GET("/peers", getPeers)
	e.GET("/peers/:id", getPeer)
	e.GET("/peers/:id/status", getPeerStatus)
	e.GET("/peers/:id/config", getPeerWireguardConfig)
	e.GET("/peers/:id/script", getPeerWireguardScript)
	e.DELETE("/peers/:id", deletePeer)

	e.POST("/access-rule/:peer_a_id/:peer_b_id", createAccessRule)
	e.GET("/access-rule/:peer_a_id/:peer_b_id", getAccessRule)
	e.DELETE("/access-rule/:peer_a_id/:peer_b_id", deleteAccessRule)

	serverAddress := os.Getenv("SERVER_ADDRESS")
	if serverAddress == "" {
		serverAddress = ":8080"
	}

	if err := e.Start(serverAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Fatal(err)
	}
	globalWaitGroup.Done()
}

func createPeer(c echo.Context) error {
	peer, err := CreatePeer()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"id":          peer.ID,
		"ip":          peer.IP,
		"public_key":  peer.PublicKey,
		"private_key": peer.PrivateKey,
		"status":      string(peer.Status),
	})
}

func getPeer(c echo.Context) error {
	id := c.Param("id")
	peer, err := GetPeer(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Peer not found",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"id":          peer.ID,
		"ip":          peer.IP,
		"public_key":  peer.PublicKey,
		"private_key": peer.PrivateKey,
		"status":      string(peer.Status),
	})
}

func getPeers(c echo.Context) error {
	peers, err := GetPeers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	response := []map[string]string{}
	for _, peer := range peers {
		response = append(response, map[string]string{
			"id":          peer.ID,
			"ip":          peer.IP,
			"public_key":  peer.PublicKey,
			"private_key": peer.PrivateKey,
			"status":      string(peer.Status),
		})
	}
	return c.JSON(http.StatusOK, response)
}

func getPeerStatus(c echo.Context) error {
	id := c.Param("id")
	status, err := GetPeerStatus(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Peer not found",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": string(status)})
}

func getPeerWireguardScript(c echo.Context) error {
	id := c.Param("id")
	peer, err := GetPeer(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, "Peer not found")
	}
	return c.JSON(http.StatusOK, peer.GenerateWireguardScript())
}

func getPeerWireguardConfig(c echo.Context) error {
	id := c.Param("id")
	peer, err := GetPeer(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Peer not found",
		})
	}
	return c.JSON(http.StatusOK, peer.GetWireguardConfig())
}

func deletePeer(c echo.Context) error {
	id := c.Param("id")
	status, err := GetPeerStatus(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	if status == PeerStatusDeleting {
		return c.NoContent(http.StatusNoContent)
	}
	// It's just logical, we don't need to delete the peer from the database
	// Worker will do their job
	err = UpdatePeerStatus(id, PeerStatusDeleting)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func createAccessRule(c echo.Context) error {
	peerAID := c.Param("peer_a_id")
	peerBID := c.Param("peer_b_id")
	rule, err := CreateAccessRule(peerAID, peerBID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"id":        rule.ID,
		"peer_a_id": rule.PeerAID,
		"peer_b_id": rule.PeerBID,
		"status":    string(rule.Status),
	})
}

func getAccessRule(c echo.Context) error {
	peerAID := c.Param("peer_a_id")
	peerBID := c.Param("peer_b_id")
	rule, err := GetAccessRule(peerAID, peerBID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Access rule not found",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"id":        rule.ID,
		"peer_a_id": rule.PeerAID,
		"peer_b_id": rule.PeerBID,
		"status":    string(rule.Status),
	})
}

func deleteAccessRule(c echo.Context) error {
	peerAID := c.Param("peer_a_id")
	peerBID := c.Param("peer_b_id")
	rule, err := GetAccessRule(peerAID, peerBID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Access rule not found",
		})
	}
	peerAIP, err := GetPeerIP(rule.PeerAID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	peerBIP, err := GetPeerIP(rule.PeerBID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	removeIptablesRuleBetweenPeers(peerAIP, peerBIP)
	err = DeleteAccessRule(rule.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.NoContent(http.StatusNoContent)
}
