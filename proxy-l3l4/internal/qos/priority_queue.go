package qos

import (
	"fmt"
	"sync"
)

// PriorityQueue implements a priority queue for packets
type PriorityQueue struct {
	mu sync.Mutex

	packets  []*Packet
	capacity int
	priority int
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(capacity, priority int) *PriorityQueue {
	return &PriorityQueue{
		packets:  make([]*Packet, 0, capacity),
		capacity: capacity,
		priority: priority,
	}
}

// Enqueue adds a packet to the queue
func (pq *PriorityQueue) Enqueue(packet *Packet) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.packets) >= pq.capacity {
		return fmt.Errorf("queue full")
	}

	pq.packets = append(pq.packets, packet)
	return nil
}

// Dequeue removes and returns the first packet
func (pq *PriorityQueue) Dequeue() *Packet {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.packets) == 0 {
		return nil
	}

	packet := pq.packets[0]
	pq.packets = pq.packets[1:]
	return packet
}

// Peek returns the first packet without removing it
func (pq *PriorityQueue) Peek() *Packet {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.packets) == 0 {
		return nil
	}

	return pq.packets[0]
}

// Depth returns the current queue depth
func (pq *PriorityQueue) Depth() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.packets)
}

// IsFull returns true if the queue is full
func (pq *PriorityQueue) IsFull() bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.packets) >= pq.capacity
}

// IsEmpty returns true if the queue is empty
func (pq *PriorityQueue) IsEmpty() bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.packets) == 0
}

// Clear clears all packets from the queue
func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.packets = make([]*Packet, 0, pq.capacity)
}
