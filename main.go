package main

import (
	"log"
	"sync"
	"time"
)

func main() {
	manager := NewNodeManager()

	bob := NewNode("bob")

	if err := manager.AddNode(bob); err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 2)

	mary := NewNode("mary")

	if err := manager.AddNode(mary); err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 2)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		time.Sleep(time.Second * 2)
		bob.Kill()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		time.Sleep(time.Second * 4)
		manager.EndDaemon()
		wg.Done()
	}()

	wg.Wait()
}
