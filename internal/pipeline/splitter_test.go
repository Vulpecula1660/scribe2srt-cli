package pipeline

import (
	"testing"
)

func TestSentenceSplitter_SplitIntoSentenceGroups_Empty(t *testing.T) {
	s := NewSentenceSplitter("en")
	groups := s.SplitIntoSentenceGroups(nil)
	if groups != nil {
		t.Errorf("expected nil for empty input, got %v", groups)
	}
}

func TestSentenceSplitter_HighPrioritySplit(t *testing.T) {
	s := NewSentenceSplitter("en")

	words := []Word{
		{Text: "Hello.", Start: 0, End: 1, Type: "word"},
		{Text: "World.", Start: 1, End: 2, Type: "word"},
	}

	groups := s.SplitIntoSentenceGroups(words)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0][0].Text != "Hello." {
		t.Errorf("first group first word = %q, want 'Hello.'", groups[0][0].Text)
	}
	if groups[1][0].Text != "World." {
		t.Errorf("second group first word = %q, want 'World.'", groups[1][0].Text)
	}
}

func TestSentenceSplitter_MediumPriorityNeedThreeWords(t *testing.T) {
	s := NewSentenceSplitter("en")

	// Only 1 accumulated word before the semicolon — should NOT split.
	words := []Word{
		{Text: "Hi;", Start: 0, End: 0.5, Type: "word"},
		{Text: "there", Start: 0.5, End: 1, Type: "word"},
	}
	groups := s.SplitIntoSentenceGroups(words)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group (medium priority, <3 words), got %d", len(groups))
	}

	// 3+ accumulated words before semicolon — should split.
	words = []Word{
		{Text: "one", Start: 0, End: 0.3, Type: "word"},
		{Text: "two", Start: 0.3, End: 0.6, Type: "word"},
		{Text: "three", Start: 0.6, End: 0.9, Type: "word"},
		{Text: "four;", Start: 0.9, End: 1.2, Type: "word"},
		{Text: "five", Start: 1.2, End: 1.5, Type: "word"},
	}
	groups = s.SplitIntoSentenceGroups(words)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (medium priority, >=3 words), got %d", len(groups))
	}
}

func TestSentenceSplitter_LowPriorityNeedsFiveWordsAndFifteenChars(t *testing.T) {
	s := NewSentenceSplitter("en")

	// 4 accumulated words — should NOT split at comma.
	words := []Word{
		{Text: "aa", Start: 0, End: 0.2, Type: "word"},
		{Text: "bb", Start: 0.2, End: 0.4, Type: "word"},
		{Text: "cc", Start: 0.4, End: 0.6, Type: "word"},
		{Text: "dd", Start: 0.6, End: 0.8, Type: "word"},
		{Text: "ee,", Start: 0.8, End: 1.0, Type: "word"},
		{Text: "ff", Start: 1.0, End: 1.2, Type: "word"},
	}
	groups := s.SplitIntoSentenceGroups(words)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group (low priority, <5 accumulated words), got %d", len(groups))
	}

	// 5+ accumulated words and 15+ chars — should split.
	words = []Word{
		{Text: "one", Start: 0, End: 0.2, Type: "word"},
		{Text: "two", Start: 0.2, End: 0.4, Type: "word"},
		{Text: "three", Start: 0.4, End: 0.6, Type: "word"},
		{Text: "four", Start: 0.6, End: 0.8, Type: "word"},
		{Text: "five", Start: 0.8, End: 1.0, Type: "word"},
		{Text: "six,", Start: 1.0, End: 1.2, Type: "word"},
		{Text: "seven", Start: 1.2, End: 1.4, Type: "word"},
	}
	groups = s.SplitIntoSentenceGroups(words)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (low priority, >=5 words & >=15 chars), got %d", len(groups))
	}
}

func TestSentenceSplitter_CJKLanguage(t *testing.T) {
	s := NewSentenceSplitter("ja")
	if !s.IsCJK {
		t.Error("expected IsCJK=true for 'ja'")
	}

	s2 := NewSentenceSplitter("zho")
	if !s2.IsCJK {
		t.Error("expected IsCJK=true for 'zho'")
	}

	s3 := NewSentenceSplitter("en")
	if s3.IsCJK {
		t.Error("expected IsCJK=false for 'en'")
	}
}

func TestSentenceSplitter_CreateBasicEntries(t *testing.T) {
	s := NewSentenceSplitter("en")

	groups := [][]Word{
		{
			{Text: "Hello ", Start: 0, End: 0.5, Type: "word"},
			{Text: "world.", Start: 0.5, End: 1.0, Type: "word"},
		},
		{
			{Text: "Goodbye.", Start: 1.5, End: 2.0, Type: "word"},
		},
	}

	entries := s.CreateBasicEntries(groups)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Text != "Hello world." {
		t.Errorf("entry[0].Text = %q, want 'Hello world.'", entries[0].Text)
	}
	if entries[0].Start != 0 || entries[0].End != 1.0 {
		t.Errorf("entry[0] timing = [%f, %f], want [0, 1.0]", entries[0].Start, entries[0].End)
	}
	if entries[0].WordCount != 2 {
		t.Errorf("entry[0].WordCount = %d, want 2", entries[0].WordCount)
	}

	if entries[1].Text != "Goodbye." {
		t.Errorf("entry[1].Text = %q, want 'Goodbye.'", entries[1].Text)
	}
}

func TestSentenceSplitter_CreateBasicEntries_SkipsEmptyGroups(t *testing.T) {
	s := NewSentenceSplitter("en")

	groups := [][]Word{
		{},
		{
			{Text: "Hello.", Start: 0, End: 1, Type: "word"},
		},
	}

	entries := s.CreateBasicEntries(groups)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (empty group skipped), got %d", len(entries))
	}
}
