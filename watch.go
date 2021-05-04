package main

import (
	"fmt"
	"time"
)

func (b *bygge) unresolve() {
	for k, tgt := range b.targets {
		tgt.resolved = false
		b.targets[k] = tgt
	}
}

func (b *bygge) waitForChange(tgt target) error {
	start := time.Now()
	for {
		lastUpdate, err := b.lastUpdated(tgt)
		if err != nil {
			return err
		}
		if lastUpdate.After(start) {
			return nil
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func (b *bygge) lastUpdated(tgt target) (time.Time, error) {

	if b.visited[tgt.name] {
		return time.Time{}, fmt.Errorf("cyclic dependency resolving %q", tgt.name)
	}
	b.visited[tgt.name] = true
	defer func() {
		b.visited[tgt.name] = false
	}()

	mostRecentUpdate := getFileDate(tgt.name)

	for _, depName := range tgt.dependencies {
		modified := time.Time{}

		dep, ok := b.targets[depName]
		if !ok {
			modified = getFileDate(depName)
		} else {
			var err error
			modified, err = b.lastUpdated(dep)
			if err != nil {
				return time.Time{}, err
			}
		}
		if modified.After(mostRecentUpdate) {
			mostRecentUpdate = modified
		}
	}

	return mostRecentUpdate, nil
}
