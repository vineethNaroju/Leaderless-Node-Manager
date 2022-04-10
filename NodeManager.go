package main

import (
	"fmt"
	"sync"
	"time"
)

type NodeManager struct {
	doneBroadcast       chan interface{}
	nodeStore           map[string]*Node
	nodeStoreMutex      *sync.RWMutex
	nodeResignedChannel chan string
	alive               bool
	closeupWg           *sync.WaitGroup
}

func NewNodeManager() *NodeManager {

	nm := &NodeManager{
		nodeStore:           make(map[string]*Node),
		nodeStoreMutex:      &sync.RWMutex{},
		doneBroadcast:       make(chan interface{}),
		nodeResignedChannel: make(chan string),
		alive:               true,
		closeupWg:           &sync.WaitGroup{},
	}

	nm.daemon()

	return nm
}

func (nm *NodeManager) AddNode(nd *Node) error {

	nm.nodeStoreMutex.Lock()
	defer nm.nodeStoreMutex.Unlock()

	if !nm.alive {
		return fmt.Errorf("manager is dead")
	}

	if _, ok := nm.nodeStore[nd.name]; ok {
		return fmt.Errorf("duplicate node " + nd.name)
	}

	nm.nodeStore[nd.name] = nd
	nd.SetResignChannel(nm.nodeResignedChannel)

	return nil
}

func (nm *NodeManager) EndDaemon() {
	nm.doneBroadcast <- struct{}{}
	nm.closeupWg.Wait()
}

func (nm *NodeManager) removeNode(resignedNodeName string) {

	nm.nodeStoreMutex.Lock()

	delete(nm.nodeStore, resignedNodeName)

	fmt.Println("Node manager removed " + resignedNodeName)

	nm.nodeStoreMutex.Unlock()
}

func (nm *NodeManager) snapshotNodeList() []*Node {

	nodes := []*Node{}

	for _, val := range nm.nodeStore {
		nodes = append(nodes, val)
	}

	return nodes
}

func (nm *NodeManager) publishNodeList() {

	nm.nodeStoreMutex.RLock()

	nodes := nm.snapshotNodeList()

	nm.nodeStoreMutex.RUnlock()

	for _, val := range nodes {
		go func(nd *Node) {
			nd.ingressNodeList <- nodes
		}(val)
	}
}

func (nm *NodeManager) killNodes() {

	nm.nodeStoreMutex.Lock()

	nm.alive = false

	nodes := nm.snapshotNodeList()

	nm.nodeStoreMutex.Unlock()

	wg := &sync.WaitGroup{}
	wg.Add(len(nodes))

	for _, val := range nodes {

		fmt.Println("killing node " + val.name)

		go func(nd *Node) {
			nd.managerKicksChannel <- struct{}{}
			close(nd.managerKicksChannel)
			wg.Done()
		}(val)
	}

	wg.Wait()
	nm.closeupWg.Done()
}

func (nm *NodeManager) publishNodeListHeartbeat(doneChan <-chan interface{}) {
	nm.closeupWg.Add(1)

	go func(done <-chan interface{}) {

		defer nm.closeupWg.Done()

		for {
			select {
			case <-done:
				return
			default:
				time.Sleep(time.Second * 1)
				nm.publishNodeList()
			}
		}
	}(doneChan)
}

func (nm *NodeManager) daemon() {

	done := make(chan interface{})

	nm.publishNodeListHeartbeat(done)

	nm.closeupWg.Add(1)

	go func(nm *NodeManager, done chan<- interface{}) {
		for {
			select {
			case resignedNodeName := <-nm.nodeResignedChannel:
				go nm.removeNode(resignedNodeName)
			case <-nm.doneBroadcast:
				done <- struct{}{}
				close(done)
				nm.killNodes()
				return
			}
		}
	}(nm, done)
}
