package main

import (
	"log"
)

type QueueJob struct {
	Type string //  peer, access_rule
	ID   string
}

var workerQueueChannel chan QueueJob

func init() {
	workerQueueChannel = make(chan QueueJob, 20000)
}

func runWorkers() {
	process()
	globalWaitGroup.Done()
}

func process() {
	for job := range workerQueueChannel {
		switch job.Type {
		case "peer":
			processPeer(job.ID)
		case "access_rule":
			processAccessRule(job.ID)
		}
	}
}

func processPeer(id string) {
	peer, err := GetPeer(id)
	if err != nil {
		log.Printf("[ERROR] Error getting peer %s: %s", id, err)
		return
	}
	switch peer.Status {
	case PeerStatusPending:
		processPeerPending(peer)
	case PeerStatusDeleting:
		processPeerDeleting(peer)
	}
}

func processPeerPending(peer *Peer) {
	addWireguardPeer(peer.PublicKey, peer.IP)
	err := UpdatePeerStatus(peer.ID, PeerStatusCreated)
	if err != nil {
		log.Printf("[ERROR] Error updating peer %s status to created: %s", peer.ID, err)
	}
}

func processPeerDeleting(peer *Peer) {
	// find out access rules that are using this peer
	accessRules, err := GetAccessRulesByPeerID(peer.ID)
	if err != nil {
		log.Printf("[ERROR] Error getting access rules for peer %s: %s", peer.ID, err)
		return
	}
	for _, accessRule := range accessRules {
		// find out other peer's ip
		if accessRule.PeerAID == peer.ID {
			peerBIP, err := GetPeerIP(accessRule.PeerBID)
			if err != nil {
				log.Printf("[ERROR] Error getting peer %s IP: %s", accessRule.PeerBID, err)
				return
			}
			removeIptablesRuleBetweenPeers(peer.IP, peerBIP)
		} else {
			peerAIP, err := GetPeerIP(accessRule.PeerAID)
			if err != nil {
				log.Printf("[ERROR] Error getting peer %s IP: %s", accessRule.PeerAID, err)
				return
			}
			removeIptablesRuleBetweenPeers(peerAIP, peer.IP)
		}
		// delete access rule
		err = DeleteAccessRule(accessRule.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting access rule %s: %s", accessRule.ID, err)
		}
	}
	removeWireguardPeer(peer.PublicKey)
	err = DeletePeer(peer.ID)
	if err != nil {
		log.Printf("[ERROR] Error deleting peer %s: %s", peer.ID, err)
	}
}

func processAccessRule(id string) {
	accessRule, err := GetAccessRuleByID(id)
	if err != nil {
		log.Printf("[ERROR] Error getting access rule %s: %s", id, err)
		return
	}
	peerAIP, err := GetPeerIP(accessRule.PeerAID)
	if err != nil {
		log.Printf("[ERROR] Error getting peer %s IP: %s", accessRule.PeerAID, err)
		return
	}
	peerBIP, err := GetPeerIP(accessRule.PeerBID)
	if err != nil {
		log.Printf("[ERROR] Error getting peer %s IP: %s", accessRule.PeerBID, err)
		return
	}
	addIptablesRuleBetweenPeers(peerAIP, peerBIP)
	err = UpdateAccessRuleStatus(accessRule.ID, AccessRuleStatusCreated)
	if err != nil {
		log.Printf("[ERROR] Error updating access rule %s status to created: %s", accessRule.ID, err)
	}
}

func queuePendingTasks() {
	pendingPeers := []Peer{}
	err := GetDB().Model(&Peer{}).Select("id").Where("status = ?", PeerStatusPending).Find(&pendingPeers).Error
	if err != nil {
		panic(err)
	}
	deletingPeers := []Peer{}
	err = GetDB().Model(&Peer{}).Select("id").Where("status = ?", PeerStatusDeleting).Find(&deletingPeers).Error
	if err != nil {
		panic(err)
	}
	pendingAccessRules := []AccessRule{}
	err = GetDB().Model(&AccessRule{}).Select("id").Where("status = ?", AccessRuleStatusPending).Find(&pendingAccessRules).Error
	if err != nil {
		panic(err)
	}

	for _, peer := range pendingPeers {
		workerQueueChannel <- QueueJob{Type: "peer", ID: peer.ID}
	}
	for _, peer := range deletingPeers {
		workerQueueChannel <- QueueJob{Type: "peer", ID: peer.ID}
	}
	for _, accessRule := range pendingAccessRules {
		workerQueueChannel <- QueueJob{Type: "access_rule", ID: accessRule.ID}
	}
}
