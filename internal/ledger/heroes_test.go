package ledger

import "testing"

func TestHeroNameKnownDotaIDs(t *testing.T) {
	cases := map[int32]string{
		1:   "Anti-Mage",
		14:  "Pudge",
		54:  "Lifestealer",
		83:  "Treant Protector",
		104: "Legion Commander",
		129: "Mars",
		136: "Marci",
		145: "Kez",
	}
	for id, want := range cases {
		if got := HeroName(id); got != want {
			t.Fatalf("HeroName(%d) = %q, want %q", id, got, want)
		}
	}
}

func TestHeroNameUnknownIDFallsBackToNumber(t *testing.T) {
	if got := HeroName(9999); got != "#9999" {
		t.Fatalf("HeroName(9999) = %q, want #9999", got)
	}
}
