package main

import (
	"fmt"
	"sync"
	"time"
)

/*
To ensure that each task is executed by one worker at a time, and no worker is idle, you can use a buffered channel with a capacity equal to the number of workers.
This channel will act as a semaphore, allowing only a fixed number of workers to execute tasks concurrently. Here's how you can achieve this:



In this code:

We define a worker function that receives tasks from a channel (taskCh) and executes them.
The main function creates a buffered channel taskCh with a capacity equal to the number of workers. This ensures that only a fixed number of tasks can be sent to workers at a time.
Worker goroutines are started, and each worker continuously receives tasks from the channel and executes them.
Tasks are generated and sent to the channel in a loop.
After sending all tasks, we close the channel to signal that no more tasks will be sent.
We wait for all workers to finish executing their tasks using wg.Wait().
This ensures that each task is executed by one worker at a time, and no worker is idle, given that the number of tasks is greater than or equal to the number of workers. Adjust numTasks and numWorkers according to your requirements.
*/

func worker(taskCh <-chan func(), wg *sync.WaitGroup, i int) {
	defer wg.Done()
	fmt.Println(" Go routine outside : ", i)
	for task := range taskCh {
		// Execute the task
		fmt.Println(" Go routine : ", i)
		time.Sleep(time.Duration(10) * time.Second)
		task()
	}
	fmt.Println(" Go routine END : ", i)
}

func main() {
	// Define the number of tasks and workers
	numTasks := 10
	numWorkers := 3

	// Create a buffered channel with capacity equal to the number of workers
	taskCh := make(chan func(), numWorkers)

	// Create a WaitGroup to wait for all tasks to finish
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(taskCh, &wg, i)
	}
	time.Sleep(time.Duration(30) * time.Second)

	// Generate tasks and send them to the channel
	for i := 0; i < numTasks; i++ {
		taskID := i + 1
		task := func() {
			fmt.Printf("Task %d executed by Worker\n", taskID)
		}
		taskCh <- task

	}

	// Close the channel to signal that no more tasks will be sent
	close(taskCh)

	// Wait for all workers to finish
	wg.Wait()
}

/*
Typed Pictorial Representation

1) go worker -> will spawn 3 goroutines and wait there why? read below
-----------------------------------------------------------------------------------------------------------
	In Go, when you range over a channel, the loop will keep waiting for values to be sent on the channel until the channel is closed.
	If the channel is not closed and there are no values being sent, the loop will block, waiting for values to arrive on the channel.
	This is the behavior you're observing.

	In your case, the for loop in the worker function is ranging over the taskCh channel:

	for task := range taskCh {
		fmt.Println("Go routine:", i)
		time.Sleep(time.Duration(1) * time.Second) // Simulate task execution time
		task()
	}


	In Go, when you range over a channel, the loop will keep waiting for values to be sent on the channel until the channel is closed.
	If the channel is not closed and there are no values being sent, the loop will block, waiting for values to arrive on the channel.
	This is the behavior you're observing.

	In your case, the for loop in the worker function is ranging over the taskCh channel:


	for task := range taskCh {
		fmt.Println("Go routine:", i)
		time.Sleep(time.Duration(1) * time.Second) // Simulate task execution time
		task()
	}
	This loop will keep waiting for functions to be sent on taskCh until the channel is closed.
	Once a function is sent on the channel, the loop will execute the received function (task).
	If there are no more values sent on the channel and the channel is not closed, the loop will block indefinitely.

	To avoid the loop blocking indefinitely, you need to ensure that the channel is eventually closed after all tasks have been sent.
	In your main function, after sending all tasks, you should close the taskCh channel:

	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)
-----------------------------------------------------------------------------------------------------------

2) Once function is put inside channel then for loop start execting the function will available workers
3) If we didn't close the channel it block taskCh for ever (NOTE)

NOTE - if didn't close the channel then we will get "all goroutines are asleep - deadlock" error

The "all goroutines are asleep - deadlock" error occurs when all goroutines in your program are blocked, and there's no possibility of progress.
In your case, this error likely arises because your worker goroutines are waiting for tasks to be sent on the taskCh channel, but the main goroutine does not close the taskCh channel after sending all tasks.
Therefore, the worker goroutines remain blocked in the loop waiting for tasks, and the program doesn't progress.


*/

/* Other Solution

package main

import (
    "sync"
)

// Task represents a unit of work.
type Task func()

// Worker represents a worker that can execute a task.
type Worker struct {
    id          int
    taskChannel chan Task
    done        chan bool
}

// NewWorker creates a new worker with the given ID.
func NewWorker(id int, taskChannel chan Task) *Worker {
    return &Worker{
        id:          id,
        taskChannel: taskChannel,
        done:        make(chan bool),
    }
}

// Start starts the worker, waiting for tasks to execute.
func (w *Worker) Start() {
    go func() {
        for {
            select {
            case task := <-w.taskChannel:
                task() // Execute task
            case <-w.done:
                return
            }
        }
    }()
}

// Stop stops the worker.
func (w *Worker) Stop() {
    w.done <- true
}

// WorkerPool represents a pool of workers.
type WorkerPool struct {
    workers []*Worker
    wg      sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
func NewWorkerPool(numWorkers int) *WorkerPool {
    wp := &WorkerPool{}
    wp.workers = make([]*Worker, numWorkers)
    for i := 0; i < numWorkers; i++ {
        wp.workers[i] = NewWorker(i, make(chan Task))
    }
    return wp
}

// Start starts all workers in the pool.
func (wp *WorkerPool) Start() {
    for _, worker := range wp.workers {
        worker.Start()
    }
}

// Stop stops all workers in the pool.
func (wp *WorkerPool) Stop() {
    for _, worker := range wp.workers {
        worker.Stop()
    }
}

// AssignTask assigns a task to an available worker.
func (wp *WorkerPool) AssignTask(task Task) {
    wp.wg.Add(1)
    go func() {
        defer wp.wg.Done()
        // Find an available worker
        for _, worker := range wp.workers {
            select {
            case worker.taskChannel <- task:
                return
            default:
            }
        }
    }()
}

// Wait waits for all tasks to be completed.
func (wp *WorkerPool) Wait() {
    wp.wg.Wait()
}

func main() {
    numTasks := 10
    numWorkers := 5

    // Create a worker pool with the specified number of workers
    pool := NewWorkerPool(numWorkers)
    pool.Start()

    // Generate some tasks
    tasks := make([]Task, numTasks)
    for i := range tasks {
        id := i // Capture the loop variable
        tasks[i] = func() {
            println("Task", id, "executed by worker", id%numWorkers)
        }
    }

    // Assign tasks to workers
    for _, task := range tasks {
        pool.AssignTask(task)
    }

    // Wait for all tasks to complete
    pool.Wait()

    // Stop all workers
    pool.Stop()
}

*/
