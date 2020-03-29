package tpcluster

import (
	"github.com/weblazy/core/logx"
	"github.com/weblazy/core/mapreduce"
	tp "github.com/weblazy/teleport"
)

type NodePush struct {
	tp.PushCtx
}

func (n *NodePush) Ping(ping *string) *tp.Status {
	sessionId := n.Session().ID()
	logx.Errorf("%s:%s", sessionId, *ping)
	return nil
}

func (n *NodePush) SendToUid(msg *Message) (int, *tp.Status) {
	sessionMap, ok := nodeInfo.uidSessions.LoadMap(msg.uid)
	if ok {
		mapreduce.MapVoid(func(source chan<- interface{}) {
			for _, session := range sessionMap {
				source <- session
			}
		}, func(item interface{}) {
			session := item.(tp.Session)
			session.Push(
				msg.path,
				msg.data,
			)
		})
	}
	return 0, nil
}
