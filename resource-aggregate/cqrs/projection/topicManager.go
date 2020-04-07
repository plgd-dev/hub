package projection

import (
	"sync"
)

type GetTopicsFunc func(string) []string

type TopicManager struct {
	topicsLock sync.Mutex
	topics     map[string]int
	getTopics  GetTopicsFunc
}

func NewTopicManager(getTopics GetTopicsFunc) *TopicManager {
	return &TopicManager{
		topics:    make(map[string]int),
		getTopics: getTopics,
	}
}

func (p *TopicManager) Add(key string) ([]string, bool) {
	var updateSubscriber bool
	var topics []string
	p.topicsLock.Lock()
	defer p.topicsLock.Unlock()
	for _, t := range p.getTopics(key) {
		if _, ok := p.topics[t]; ok {
			p.topics[t]++
		} else {
			updateSubscriber = true
			p.topics[t] = 1
		}
	}
	if updateSubscriber {
		for t := range p.topics {
			topics = append(topics, t)
		}
	}
	return topics, updateSubscriber
}

func (p *TopicManager) Remove(key string) ([]string, bool) {
	var updateSubscriber bool
	var topics []string
	p.topicsLock.Lock()
	defer p.topicsLock.Unlock()
	for _, t := range p.getTopics(key) {
		if _, ok := p.topics[t]; ok {
			p.topics[t]--
			if p.topics[t] <= 0 {
				delete(p.topics, t)
				updateSubscriber = true
			}
		}
	}
	if updateSubscriber {
		for t := range p.topics {
			topics = append(topics, t)
		}
	}
	return topics, updateSubscriber
}
