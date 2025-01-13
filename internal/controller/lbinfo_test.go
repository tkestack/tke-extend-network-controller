package controller

import "testing"

func TestLbInfo(t *testing.T) {
	lbId := "lb-3npjk3wh/BGP,lb-b6ukw5nr/DianXing"
	lbInfos := getLbInfos(lbId)
	if lbInfos[0].LbId != "lb-3npjk3wh" || lbInfos[0].Alias != "BGP" {
		t.Errorf("bad lbInfo: %v", lbInfos[0])
	}
	if lbInfos[1].LbId != "lb-b6ukw5nr" || lbInfos[1].Alias != "DianXing" {
		t.Errorf("bad lbInfo: %v", lbInfos[1])
	}
}
