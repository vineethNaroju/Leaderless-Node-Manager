package main

import "fmt"

type Node struct {
	ingressNodeList     chan []*Node
	name                string
	killChannel         chan interface{}
	nodes               []*Node
	resignationChannel  chan<- string
	managerKicksChannel chan interface{}
}

func NewNode(name string) *Node {
	nd := &Node{
		ingressNodeList:     make(chan []*Node),
		name:                name,
		killChannel:         make(chan interface{}),
		managerKicksChannel: make(chan interface{}),
	}

	nd.daemon()

	return nd
}

func (nd *Node) SetResignChannel(nodeResignedChannel chan<- string) {
	nd.resignationChannel = nodeResignedChannel
}

func (nd *Node) Kill() {
	nd.killChannel <- struct{}{}
}

func (nd *Node) updateNodeList(nodeList []*Node) {
	nd.nodes = nodeList
	fmt.Println("Node " + nd.name + " received updated node list size " + fmt.Sprint(len(nodeList)))
}

func (nd *Node) daemon() {
	go func() {
		for {
			select {
			case <-nd.managerKicksChannel:
				fmt.Println("Node " + nd.name + " got kicked by manager")
				return
			case nodeList := <-nd.ingressNodeList:
				nd.updateNodeList(nodeList)
			case <-nd.killChannel:
				fmt.Println("Node " + nd.name + " is killed by user")
				nd.resignationChannel <- nd.name
				return
			}
		}
	}()
}
