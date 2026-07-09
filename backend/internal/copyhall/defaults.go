package copyhall



import (

	"fmt"



	"caipiao/backend/internal/schemes"

)



type defaultSlot struct {

	Rank       int

	SchemeName string

	PlayMethod string

	PlayTypeID string

	SubPlayID  string

}



// lotteryCatalog 跟单榜默认展示彩种（新 47 子集，purge 后无旧码）。

var lotteryCatalog = []struct {

	Code  string

	Label string

}{

	{Code: "tron_ffc_1m", Label: "波场1分彩"},

	{Code: "hash_ffc_1m", Label: "哈希1分彩"},

	{Code: "eth_ffc_1m", Label: "以太坊1分彩"},

	{Code: "bnb_ffc_1m", Label: "币安1分彩"},

	{Code: "tron_jisu", Label: "波场极速彩"},

	{Code: "tron_syxw", Label: "波场11选5"},

}



var masterDefaultSlots = []defaultSlot{

	{1, "太乙后二", "定位胆万位", "dingwei", "dingwei_wan"},

	{2, "紫燕万位", "定位胆后二", "hou2", "hou2_zhixuan_fs"},

	{3, "莺凤十位", "定位胆十位", "dingwei", "dingwei_shi"},

	{4, "宛天个位", "定位胆个位", "dingwei", "dingwei_ge"},

	{5, "路线6000+", "组选六", "zhong3", "zhong3_zu6"},

	{6, "打狗前二", "定位胆前三", "qian3", "qian3_zhixuan_fs"},

	{7, "邯肖任四", "任选四", "renxuan", "ren4_zu24"},

	{8, "关冲70+", "定位胆后一", "dingwei", "dingwei_ge"},

	{9, "猎豹后二", "定位胆千位", "dingwei", "dingwei_qian"},

	{10, "青衫万位", "定位胆任二", "renxuan", "ren2_zhixuan_fs"},

}



var contraryDefaultSlots = []defaultSlot{

	{1, "逆锋万位", "定位胆万位", "dingwei", "dingwei_wan"},

	{2, "反打后二", "定位胆后二", "hou2", "hou2_zhixuan_fs"},

	{3, "折戟十位", "定位胆十位", "dingwei", "dingwei_shi"},

	{4, "回风个位", "定位胆个位", "dingwei", "dingwei_ge"},

	{5, "暗线3000-", "组选六", "zhong3", "zhong3_zu6"},

	{6, "退守前三", "定位胆前三", "qian3", "qian3_zhixuan_fs"},

	{7, "虚晃任四", "任选四", "renxuan", "ren4_zu24"},

	{8, "蛰伏50-", "定位胆后一", "dingwei", "dingwei_ge"},

	{9, "裂空后一", "定位胆千位", "dingwei", "dingwei_qian"},

	{10, "寒江千位", "定位胆任二", "renxuan", "ren2_zhixuan_fs"},

}



func LotteryLabel(code string) string {

	for _, lot := range lotteryCatalog {

		if lot.Code == code {

			return lot.Label

		}

	}

	return code

}



func defaultSchemeID(boardKind string, rank int) string {

	prefix := "copy_demo"

	if boardKind == "contrary" {

		prefix = "copy_contrary"

	}

	if rank == 1 {

		return prefix + "_3001"

	}

	return fmt.Sprintf("%s_%d", prefix, 3000+rank)

}



func defaultBoardSlots(boardKind string) []RankSlot {

	src := masterDefaultSlots

	if boardKind == "contrary" {

		src = contraryDefaultSlots

	}

	slots := make([]RankSlot, 0, len(src))

	for _, row := range src {

		playTypeID := row.PlayTypeID

		subPlayID := row.SubPlayID

		if playTypeID == "" {

			playTypeID, subPlayID = schemes.PlayIDsFromMethod(row.PlayMethod)

		}

		slots = append(slots, RankSlot{

			Rank: row.Rank,

			LotteryCode: defaultGlobalLotteryCode,

			SchemeID:   defaultSchemeID(boardKind, row.Rank),

			SchemeName: row.SchemeName,

			PlayMethod: row.PlayMethod,

			PlayTypeID: playTypeID,

			SubPlayID:  subPlayID,

		})

	}

	return slots

}



func defaultRankingsState() AdminRankingsState {
	return AdminRankingsState{
		Master:   defaultBoardSlots("master"),
		Contrary: defaultBoardSlots("contrary"),
	}
}

