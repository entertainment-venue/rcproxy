// Copyright (c) 2021 Andy Pan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build (freebsd || dragonfly || darwin) && poll_opt
// +build freebsd dragonfly darwin
// +build poll_opt

package netpoll

import (
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"

	"rcproxy/core/internal/queue"
	"rcproxy/core/pkg/errors"
	"rcproxy/core/pkg/logging"
)

// Poller represents a poller which is in charge of monitoring file-descriptors.
type Poller struct {
	fd                   int
	wakeupCall           int32
	asyncTaskQueue       queue.AsyncTaskQueue // queue with low priority
	urgentAsyncTaskQueue queue.AsyncTaskQueue // queue with high priority
}

// OpenPoller instantiates a poller.
func OpenPoller() (poller *Poller, err error) {
	poller = new(Poller)
	if poller.fd, err = unix.Kqueue(); err != nil {
		poller = nil
		err = os.NewSyscallError("kqueue", err)
		return
	}
	if _, err = unix.Kevent(poller.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("kevent add|clear", err)
		return
	}
	poller.asyncTaskQueue = queue.NewLockFreeQueue()
	poller.urgentAsyncTaskQueue = queue.NewLockFreeQueue()
	return
}

// Close closes the poller.
func (p *Poller) Close() error {
	return os.NewSyscallError("close", unix.Close(p.fd))
}

var note = []unix.Kevent_t{{
	Ident:  0,
	Filter: unix.EVFILT_USER,
	Fflags: unix.NOTE_TRIGGER,
}}

// UrgentTrigger puts task into urgentAsyncTaskQueue and wakes up the poller which is waiting for network-events,
// then the poller will get tasks from urgentAsyncTaskQueue and run them.
//
// Note that urgentAsyncTaskQueue is a queue with high-priority and its size is expected to be small,
// so only those urgent tasks should be put into this queue.
func (p *Poller) UrgentTrigger(fn queue.TaskFunc, arg interface{}) (err error) {
	task := queue.GetTask()
	task.Run, task.Arg = fn, arg
	p.urgentAsyncTaskQueue.Enqueue(task)
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		if _, err = unix.Kevent(p.fd, note, nil, nil); err == unix.EAGAIN {
			err = nil
		}
	}
	return os.NewSyscallError("kevent trigger", err)
}

// Trigger is like UrgentTrigger but it puts task into asyncTaskQueue,
// call this method when the task is not so urgent, for instance writing data back to the peer.
//
// Note that asyncTaskQueue is a queue with low-priority whose size may grow large and tasks in it may backlog.
func (p *Poller) Trigger(fn queue.TaskFunc, arg interface{}) (err error) {
	task := queue.GetTask()
	task.Run, task.Arg = fn, arg
	p.asyncTaskQueue.Enqueue(task)
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		if _, err = unix.Kevent(p.fd, note, nil, nil); err == unix.EAGAIN {
			err = nil
		}
	}
	return os.NewSyscallError("kevent trigger", err)
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling(trick func(), msgTimeout func()) error {
	el := newEventList(InitPollEventsCap)

	var (
		doChores bool
	)
	for {
		trick()
		n, err := unix.Kevent(p.fd, nil, el.events, &unix.Timespec{Sec: 0, Nsec: int64(200 * time.Millisecond)})
		if n == 0 || (n < 0 && err == unix.EINTR) {
			runtime.Gosched()
			continue
		} else if err != nil {
			logging.Errorf("error occurs in kqueue: %v", os.NewSyscallError("kevent wait", err))
			return err
		}
		logging.Debugf("event loop trigger")

		var evFilter int16
		for i := 0; i < n; i++ {
			ev := &el.events[i]
			if ev.Ident != 0 {
				evFilter = ev.Filter
				if (ev.Flags&unix.EV_EOF != 0) || (ev.Flags&unix.EV_ERROR != 0) {
					evFilter = EVFilterSock
				}
				pollAttachment := (*PollAttachment)(unsafe.Pointer(ev.Udata))
				switch err = pollAttachment.Callback(int(ev.Ident), evFilter); err {
				case nil:
				case errors.ErrAcceptSocket, errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in event-loop: %v", err)
				}
			} else { // poller is awakened to run tasks in queues.
				doChores = true
			}
		}

		if doChores {
			doChores = false
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			for i := 0; i < MaxAsyncTasksAtOneTime; i++ {
				if task = p.asyncTaskQueue.Dequeue(); task == nil {
					break
				}
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			atomic.StoreInt32(&p.wakeupCall, 0)
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) && atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
				switch _, err = unix.Kevent(p.fd, note, nil, nil); err {
				case nil, unix.EAGAIN:
				default:
					doChores = true
				}
			}
		}

		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
		msgTimeout()
	}
}

// AddReadWrite registers the given file-descriptor with readable and writable events to the poller.
func (p *Poller) AddReadWrite(pa *PollAttachment) error {
	var evs [2]unix.Kevent_t
	evs[0].Ident = uint64(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	evs[1] = evs[0]
	evs[1].Filter = unix.EVFILT_WRITE
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = uint64(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = uint64(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = uint64(pa.FD)
	evs[0].Flags = unix.EV_DELETE
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent delete", err)
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(pa *PollAttachment) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = uint64(pa.FD)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(_ int) error {
	return nil
}
