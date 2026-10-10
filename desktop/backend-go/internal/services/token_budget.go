package services

import (
	"strings"
)

// BudgetItem is a unit of context that can be kept or evicted.
//
// Order in the slice is treated as recency (LRU): earlier items are evicted first
// when priorities are equal.
//
// TokenCount is an estimate; if 0 it will be computed from Content.
// Priority: higher means more important.
// Pinned: never evict (will be truncated as a last resort if pinned alone exceeds budget).
type BudgetItem struct {
	Key        string
	Type       string
	Content    string
	TokenCount int
	Priority   int
	Pinned     bool
}

type BudgetResult struct {
	Kept         []BudgetItem
	Evicted      []BudgetItem
	UsedTokens   int
	BudgetTokens int
}

// ApplyTokenBudget evicts items until the total estimated tokens fit within maxTokens.
// Eviction order is: lowest Priority first; if tied, LRU (earlier in slice) first.
func ApplyTokenBudget(items []BudgetItem, maxTokens int) BudgetResult {
	if maxTokens <= 0 {
		used := 0
		for i := range items {
			items[i] = ensureTokenCount(items[i])
			used += items[i].TokenCount
		}
		return BudgetResult{Kept: items, UsedTokens: used, BudgetTokens: maxTokens}
	}

	kept := make([]BudgetItem, 0, len(items))
	for _, it := range items {
		kept = append(kept, ensureTokenCount(it))
	}

	used := sumTokens(kept)

	// If we're over budget, evict non-pinned items by (priority asc, index asc).
	// Repeat until within budget or no evictable items remain.
	for used > maxTokens {
		idx := pickEvictionIndex(kept)
		if idx < 0 {
			break
		}
		used -= kept[idx].TokenCount
		kept = append(kept[:idx], kept[idx+1:]...)
	}

	evicted := make([]BudgetItem, 0)
	if len(kept) < len(items) {
		// Reconstruct evicted list by diffing keys+content.
		keptMap := make(map[string]int, len(kept))
		for _, it := range kept {
			keptMap[itemFingerprint(it)]++
		}
		for _, it := range items {
			fp := itemFingerprint(it)
			if keptMap[fp] > 0 {
				keptMap[fp]--
				continue
			}
			evicted = append(evicted, ensureTokenCount(it))
		}
	}

	// Last resort: if pinned content alone exceeds budget, truncate the largest pinned item.
	if used > maxTokens {
		used, kept = forceTruncatePinnedToBudget(kept, maxTokens)
	}

	return BudgetResult{
		Kept:         kept,
		Evicted:      evicted,
		UsedTokens:   used,
		BudgetTokens: maxTokens,
	}
}

func ensureTokenCount(it BudgetItem) BudgetItem {
	if it.TokenCount <= 0 {
		it.TokenCount = EstimateTokens(it.Content)
	}
	return it
}

func sumTokens(items []BudgetItem) int {
	t := 0
	for _, it := range items {
		t += it.TokenCount
	}
	return t
}

func pickEvictionIndex(items []BudgetItem) int {
	bestIdx := -1
	bestPriority := 0
	for i, it := range items {
		if it.Pinned {
			continue
		}
		if bestIdx == -1 {
			bestIdx = i
			bestPriority = it.Priority
			continue
		}
		// Lower priority evicts first.
		if it.Priority < bestPriority {
			bestIdx = i
			bestPriority = it.Priority
			continue
		}
		// If equal, keep the earlier one as the eviction candidate (LRU) => no change.
	}
	return bestIdx
}

func forceTruncatePinnedToBudget(items []BudgetItem, budget int) (int, []BudgetItem) {
	if budget <= 0 {
		return sumTokens(items), items
	}

	used := sumTokens(items)
	if used <= budget {
		return used, items
	}

	// Find largest pinned item (by token count) to truncate.
	idx := -1
	maxTokens := -1
	for i, it := range items {
		if !it.Pinned {
			continue
		}
		if it.TokenCount > maxTokens {
			idx = i
			maxTokens = it.TokenCount
		}
	}
	if idx < 0 {
		return used, items
	}

	// Truncate content proportionally by characters.
	toRemove := used - budget
	if toRemove <= 0 {
		return used, items
	}

	// Estimate characters per token ~ 4.
	trimChars := toRemove * 4
	content := items[idx].Content
	if trimChars >= len(content) {
		items[idx].Content = "[context truncated to fit token budget]"
	} else {
		items[idx].Content = strings.TrimRight(content[:len(content)-trimChars], " \n\t") + "\n… (context truncated)"
	}
	items[idx].TokenCount = EstimateTokens(items[idx].Content)
	used = sumTokens(items)
	return used, items
}

func itemFingerprint(it BudgetItem) string {
	return it.Key + "\n" + it.Type + "\n" + it.Content
}
