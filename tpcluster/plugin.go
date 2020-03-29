package tpcluster

import (
	"github.com/weblazy/teleport"
	"lazygo/core/database/redis"
)

type (
	postAcceptPlugin struct {
		tp.PostAcceptPlugin
	}

	postDisconnectPlugin struct {
		tp.PostDisconnectPlugin
	}
)

const (
	userPrefix  = "user#"
	groupPrefix = "group#"
)

func (p *postAcceptPlugin) Name() string {
	return "OnClientConnect"
}

func (p *postAcceptPlugin) PostAccept(session tp.PreSession) *tp.Status {
	// id := session.ID()
	// nodeInfo.cHashRing.Get()
	return nil
}

func (p *postDisconnectPlugin) Name() string {
	return "OnClientDisConnect"
}

func (p *postDisconnectPlugin) PostDisconnect(session tp.BaseSession) *tp.Status {
	sid := session.ID()
	node := nodeInfo.userHashRing.Get(sid)
	if uid := session.LoadUid(); uid != "" {
		node.Extra.(*redis.Redis).Hdel(userPrefix+uid, nodeInfo.transAddress)
	}
	return nil
}
