package main



type WorkQueue struct {
	queue	chan interface{}
}

func (w *WorkQueue) Enqueue(item interface{}) {
	w.queue <- item
}

func (w *WorkQueue) Dequeue() (interface{}, bool) {
	i, ok := <-w.queue
	return i, ok
}

func (w *WorkQueue) ShutDown() {
	close(w.queue)
}

func NewWorkQueue() WorkQueue {
	wq := WorkQueue{
		queue: make(chan interface{}),
	}
	return wq
}