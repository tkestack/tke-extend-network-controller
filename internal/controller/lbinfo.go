package controller

import (
	"fmt"
	"strings"
)

type LBInfo struct {
	LbId  string
	Alias string
}

type ListenerInfo struct {
	LbName     string
	ListenerId string
}

func (li ListenerInfo) String() string {
	if li.LbName != "" {
		return fmt.Sprintf("%s/%s", li.ListenerId, li.LbName)
	}
	return li.ListenerId
}

type ListenerInfos []ListenerInfo

func (ls ListenerInfos) String() string {
	if len(ls) == 1 {
		return ls[0].ListenerId
	}
	ss := []string{}
	for _, l := range ls {
		ss = append(ss, l.String())
	}
	return strings.Join(ss, ",")
}

func (i *LBInfo) Name() string {
	if i.Alias != "" {
		return i.Alias
	}
	return i.LbId
}

func getListenerInfos(listenerId string) (ls ListenerInfos) {
	ss := strings.Split(listenerId, ",")
	for _, s := range ss {
		idAndName := strings.Split(s, "/")
		id := idAndName[0]
		name := ""
		if len(idAndName) == 2 {
			name = idAndName[1]
		}
		ls = append(ls, ListenerInfo{ListenerId: id, LbName: name})
	}
	return
}

func getLbInfos(lbId string) (lbInfos []LBInfo) {
	ss := strings.Split(lbId, ",")
	for _, s := range ss {
		idAndName := strings.Split(s, "/")
		id := idAndName[0]
		name := ""
		if len(idAndName) == 2 {
			name = idAndName[1]
		}
		lbInfos = append(lbInfos, LBInfo{LbId: id, Alias: name})
	}
	return
}

func getLbIds(lbId string) []string {
	ss := strings.Split(lbId, ",")
	if len(ss) > 1 {
		ids := []string{}
		for _, s := range ss {
			ids = append(ids, strings.Split(s, "/")[0])
		}
		return ids
	} else {
		return ss
	}
}
