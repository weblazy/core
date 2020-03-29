package collection

import (
	"container/list"
	"fmt"
	"lazygo/core/threading"
	"lazygo/core/timex"
	"time"
)

type (
	Execute func(key, value interface{})

	TimingWheel struct {
		interval      time.Duration
		ticker        timex.Ticker
		slots         []*list.List
		timers        map[interface{}]*positionEntry
		tickedPos     int
		numSlots      int
		execute       Execute
		setChannel    chan timingEntry
		moveChannel   chan baseEntry
		removeChannel chan interface{}
		stopChannel   chan struct{}
	}

	timingEntry struct {
		baseEntry
		value   interface{}
		circle  int
		diff    int
		removed bool
	}

	baseEntry struct {
		delay time.Duration
		key   interface{}
	}

	positionEntry struct {
		pos  int
		item *timingEntry
	}

	timingTask struct {
		key   interface{}
		value interface{}
	}
)

func NewTimingWheel(interval time.Duration, numSlots int, execute Execute) (*TimingWheel, error) {
	if interval <= 0 || numSlots <= 0 || execute == nil {
		return nil, fmt.Errorf("interval: %v, slots: %d, execute: %p", interval, numSlots, execute)
	}

	return newTimingWheelWithClock(interval, numSlots, execute, timex.NewRealTicker(interval))
}

func newTimingWheelWithClock(interval time.Duration, numSlots int, execute Execute, ticker timex.Ticker) (
	*TimingWheel, error) {
	tw := &TimingWheel{
		interval:      interval,
		ticker:        ticker,
		slots:         make([]*list.List, numSlots),
		timers:        make(map[interface{}]*positionEntry),
		tickedPos:     -1,
		execute:       execute,
		numSlots:      numSlots,
		setChannel:    make(chan timingEntry),
		moveChannel:   make(chan baseEntry),
		removeChannel: make(chan interface{}),
		stopChannel:   make(chan struct{}),
	}

	tw.initSlots()
	go tw.run()

	return tw, nil
}

func (tw *TimingWheel) MoveTimer(key interface{}, delay time.Duration) {
	if delay <= 0 || key == nil {
		return
	}

	tw.moveChannel <- baseEntry{
		delay: delay,
		key:   key,
	}
}

func (tw *TimingWheel) RemoveTimer(key interface{}) {
	if key == nil {
		return
	}

	tw.removeChannel <- key
}

func (tw *TimingWheel) SetTimer(key, value interface{}, delay time.Duration) {
	if delay <= 0 || key == nil {
		return
	}

	tw.setChannel <- timingEntry{
		baseEntry: baseEntry{
			delay: delay,
			key:   key,
		},
		value: value,
	}
}

func (tw *TimingWheel) Stop() {
	close(tw.stopChannel)
}

func (tw *TimingWheel) getPositionAndCircle(d time.Duration) (pos int, circle int) {
	delay := int(d / tw.interval)
	pos = (tw.tickedPos + delay) % tw.numSlots
	circle = delay / tw.numSlots

	return
}

func (tw *TimingWheel) initSlots() {
	for i := 0; i < tw.numSlots; i++ {
		tw.slots[i] = list.New()
	}
}

func (tw *TimingWheel) moveTask(task baseEntry) {
	timer, ok := tw.timers[task.key]
	if !ok {
		return
	}

	if task.delay < tw.interval {
		threading.GoSafe(func() {
			tw.execute(timer.item.key, timer.item.value)
		})
		return
	}

	pos, circle := tw.getPositionAndCircle(task.delay)
	item := timer.item
	// use: (tw.numSlots + x) % tw.numSlots
	// to avoid diff be negative
	// don't care about the old diff, because we only care about the new position diff
	oldSteps := (tw.numSlots + timer.pos - tw.tickedPos) % tw.numSlots
	newSteps := (tw.numSlots + pos - tw.tickedPos) % tw.numSlots
	if newSteps < oldSteps {
		if circle > 0 {
			item.circle = circle - 1
			item.diff = (tw.numSlots + pos - timer.pos) % tw.numSlots
		} else {
			item.removed = true
			newItem := &timingEntry{
				baseEntry: task,
				value:     item.value,
			}
			tw.slots[pos].PushBack(newItem)
			tw.setTimerPosition(pos, newItem)
		}
	} else {
		item.circle = circle
		item.diff = (tw.numSlots + pos - timer.pos) % tw.numSlots
	}
}

func (tw *TimingWheel) onTick() {
	tw.tickedPos = (tw.tickedPos + 1) % tw.numSlots
	l := tw.slots[tw.tickedPos]
	tw.scanAndRunTasks(l)
}

func (tw *TimingWheel) removeTask(key interface{}) {
	timer, ok := tw.timers[key]
	if !ok {
		return
	}

	timer.item.removed = true
}

func (tw *TimingWheel) run() {
	for {
		select {
		case <-tw.ticker.Chan():
			tw.onTick()
		case task := <-tw.setChannel:
			tw.setTask(&task)
		case key := <-tw.removeChannel:
			tw.removeTask(key)
		case task := <-tw.moveChannel:
			tw.moveTask(task)
		case <-tw.stopChannel:
			tw.ticker.Stop()
			return
		}
	}
}

func (tw *TimingWheel) runTasks(tasks []timingTask) {
	if len(tasks) == 0 {
		return
	}

	go func() {
		for i := range tasks {
			threading.RunSafe(func() {
				tw.execute(tasks[i].key, tasks[i].value)
			})
		}
	}()
}

func (tw *TimingWheel) scanAndRunTasks(l *list.List) {
	var tasks []timingTask

	for e := l.Front(); e != nil; {
		task := e.Value.(*timingEntry)
		if task.removed {
			next := e.Next()
			l.Remove(e)
			delete(tw.timers, task.key)
			e = next
			continue
		} else if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		} else if task.diff > 0 {
			next := e.Next()
			l.Remove(e)
			// (tw.tickedPos+task.diff)%tw.numSlots
			// cannot be the same value of tw.tickedPos
			pos := (tw.tickedPos + task.diff) % tw.numSlots
			tw.slots[pos].PushBack(task)
			tw.setTimerPosition(pos, task)
			task.diff = 0
			e = next
			continue
		}

		tasks = append(tasks, timingTask{
			key:   task.key,
			value: task.value,
		})
		next := e.Next()
		l.Remove(e)
		delete(tw.timers, task.key)
		e = next
	}

	tw.runTasks(tasks)
}

func (tw *TimingWheel) setTask(task *timingEntry) {
	if task.delay < tw.interval {
		task.delay = tw.interval
	}

	if entry, ok := tw.timers[task.key]; ok {
		entry.item.value = task.value
		tw.moveTask(task.baseEntry)
	} else {
		pos, circle := tw.getPositionAndCircle(task.delay)
		task.circle = circle
		tw.slots[pos].PushBack(task)
		tw.setTimerPosition(pos, task)
	}
}

func (tw *TimingWheel) setTimerPosition(pos int, task *timingEntry) {
	if timer, ok := tw.timers[task.key]; ok {
		timer.pos = pos
	} else {
		tw.timers[task.key] = &positionEntry{
			pos:  pos,
			item: task,
		}
	}
}
