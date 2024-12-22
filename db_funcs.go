package main

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetPeer(peerID string) (*Peer, error) {
	var peer Peer
	err := GetDB().First(&peer, "id = ?", peerID).Error
	return &peer, err
}

func GetPeerIP(peerID string) (string, error) {
	var peer Peer
	err := GetDB().First(&peer, "id = ?", peerID).Select("ip").Error
	return peer.IP, err
}

func GetPeerStatus(peerID string) (PeerStatus, error) {
	var peer Peer
	err := GetDB().First(&peer, "id = ?", peerID).Select("status").Error
	return peer.Status, err
}

func GetPeers() ([]Peer, error) {
	var peers []Peer
	err := GetDB().Find(&peers).Error
	return peers, err
}

func CreatePeer() (*Peer, error) {
	ip := getUniqueIPInSubnet()
	privateKey, err := generateWireguardPrivateKey()
	if err != nil {
		return nil, err
	}
	publicKey, err := generateWireguardPublicKey(privateKey)
	if err != nil {
		return nil, err
	}
	peer := &Peer{
		ID:         uuid.New().String(),
		IP:         ip,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Status:     PeerStatusPending,
	}
	err = GetDB().Create(peer).Error
	if err == nil {
		workerQueueChannel <- QueueJob{Type: "peer", ID: peer.ID}
	}
	return peer, err
}

func UpdatePeerStatus(peerID string, status PeerStatus) error {
	err := GetDB().Model(&Peer{}).Where("id = ?", peerID).Update("status", status).Error
	if err == nil && status == PeerStatusDeleting {
		workerQueueChannel <- QueueJob{Type: "peer", ID: peerID}
	}
	return err
}

func DeletePeer(peerID string) error {
	return GetDB().Delete(&Peer{}, "id = ?", peerID).Error
}

func GetAccessRuleByID(id string) (*AccessRule, error) {
	var accessRule AccessRule
	err := GetDB().First(&accessRule, "id = ?", id).Error
	return &accessRule, err
}

func GetAccessRule(peerAID, peerBID string) (*AccessRule, error) {
	var accessRule AccessRule
	err := GetDB().First(&accessRule, "peer_a_id = ? AND peer_b_id = ?", peerAID, peerBID).Error
	if err == nil {
		return &accessRule, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// try out peerBID -> peerAID
	err = GetDB().First(&accessRule, "peer_b_id = ? AND peer_a_id = ?", peerAID, peerBID).Error
	if err != nil {
		return nil, err
	}
	return &accessRule, nil
}

func GetAccessRulesByPeerID(peerID string) ([]AccessRule, error) {
	var accessRules []AccessRule
	err := GetDB().Find(&accessRules, "peer_a_id = ? OR peer_b_id = ?", peerID, peerID).Error
	return accessRules, err
}

func IsAccessRuleExist(peerAID, peerBID string) (bool, error) {
	accessRule, err := GetAccessRule(peerAID, peerBID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return accessRule != nil, nil
}

func CreateAccessRule(peerAID, peerBID string) (*AccessRule, error) {
	peerAID = strings.TrimSpace(peerAID)
	peerBID = strings.TrimSpace(peerBID)
	isExist, err := IsAccessRuleExist(peerAID, peerBID)
	if err != nil {
		return nil, err
	}
	if isExist {
		record, err := GetAccessRule(peerAID, peerBID)
		if err != nil {
			return nil, errors.New("failed to get access rule " + peerAID + " -> " + peerBID + " : " + err.Error())
		}
		return record, nil
	}
	// Validate peerAID and peerBID
	_, err = GetPeer(peerAID)
	if err != nil {
		return nil, errors.New("failed to get peer " + peerAID + " : " + err.Error())
	}
	_, err = GetPeer(peerBID)
	if err != nil {
		return nil, errors.New("failed to get peer " + peerBID + " : " + err.Error())
	}
	// check if peerAID and peerBID are the same
	if peerAID == peerBID {
		return nil, errors.New("peerAID and peerBID cannot be the same")
	}
	accessRule := &AccessRule{
		ID:      uuid.New().String(),
		PeerAID: peerAID,
		PeerBID: peerBID,
		Status:  AccessRuleStatusPending,
	}
	err = GetDB().Create(accessRule).Error
	if err == nil {
		workerQueueChannel <- QueueJob{Type: "access_rule", ID: accessRule.ID}
	}
	return accessRule, err
}

func UpdateAccessRuleStatus(ruleID string, status AccessRuleStatus) error {
	return GetDB().Model(&AccessRule{}).Where("id = ?", ruleID).Update("status", status).Error
}

func DeleteAccessRule(ruleID string) error {
	return GetDB().Delete(&AccessRule{}, "id = ?", ruleID).Error
}
