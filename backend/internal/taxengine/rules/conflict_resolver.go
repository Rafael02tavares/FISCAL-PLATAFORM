package rules

import (
	"sort"
	"strings"
)

type ConflictResolver struct{}

func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

func (r *ConflictResolver) Resolve(matches []*SelectedRule) *SelectedRule {
	if len(matches) == 0 {
		return nil
	}

	filtered := make([]*SelectedRule, 0, len(matches))
	for _, match := range matches {
		if match == nil {
			continue
		}
		filtered = append(filtered, match)
	}

	if len(filtered) == 0 {
		return nil
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i]
		right := filtered[j]

		if left.MatchedRule.Score != right.MatchedRule.Score {
			return left.MatchedRule.Score > right.MatchedRule.Score
		}

		if left.Rule.Priority != right.Rule.Priority {
			return left.Rule.Priority > right.Rule.Priority
		}

		if left.Rule.SpecificityHint != right.Rule.SpecificityHint {
			return left.Rule.SpecificityHint > right.Rule.SpecificityHint
		}

		leftValidFrom := left.Rule.ValidFrom.Unix()
		rightValidFrom := right.Rule.ValidFrom.Unix()
		if leftValidFrom != rightValidFrom {
			return leftValidFrom > rightValidFrom
		}

		leftCode := strings.TrimSpace(left.Rule.Code)
		rightCode := strings.TrimSpace(right.Rule.Code)
		if leftCode != rightCode {
			return leftCode < rightCode
		}

		return left.Rule.ID < right.Rule.ID
	})

	return filtered[0]
}