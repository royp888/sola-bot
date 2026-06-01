package bot

import (
	"strings"
	"testing"

	"github.com/dabowin/sola/internal/api"
)

func TestLotteryJoinButtonVisibilityByJoinType(t *testing.T) {
	cases := []struct {
		name     string
		joinType string
		want     bool
	}{
		{name: "button lottery", joinType: "button", want: true},
		{name: "keyword lottery", joinType: "keyword", want: false},
		{name: "button and keyword lottery", joinType: "both", want: true},
		{name: "legacy lottery", joinType: "", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lotteryHasJoinButton(tc.joinType); got != tc.want {
				t.Fatalf("lotteryHasJoinButton(%q) = %v, want %v", tc.joinType, got, tc.want)
			}
		})
	}
}

func TestLotteryAnnouncementTextSeparatesButtonAndKeyword(t *testing.T) {
	buttonText := lotteryAnnouncementText(api.Lottery{
		ID:          1,
		Title:       "button draw",
		Prize:       "coupon",
		WinnerCount: 1,
		JoinType:    "button",
	})
	if !strings.Contains(buttonText, "按钮抽奖活动 #1") {
		t.Fatalf("button announcement missing button label: %s", buttonText)
	}
	if !strings.Contains(buttonText, "点击下方按钮参与抽奖。") {
		t.Fatalf("button announcement missing button instruction: %s", buttonText)
	}
	if strings.Contains(buttonText, "口令：") || strings.Contains(buttonText, "发送口令") {
		t.Fatalf("button announcement should not mention keyword: %s", buttonText)
	}

	keywordText := lotteryAnnouncementText(api.Lottery{
		ID:          2,
		Title:       "keyword draw",
		Prize:       "coupon",
		WinnerCount: 1,
		JoinType:    "keyword",
		JoinKeyword: "888",
	})
	if !strings.Contains(keywordText, "口令抽奖活动 #2") {
		t.Fatalf("keyword announcement missing keyword label: %s", keywordText)
	}
	if !strings.Contains(keywordText, "口令：888") || !strings.Contains(keywordText, "发送口令「888」参与抽奖。") {
		t.Fatalf("keyword announcement missing keyword instruction: %s", keywordText)
	}
	if strings.Contains(keywordText, "点击下方按钮参与抽奖。") {
		t.Fatalf("keyword announcement should not use button-only instruction: %s", keywordText)
	}
}
