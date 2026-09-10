package matching

import (
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

// dedupeDraft ensures uniqueness by the participants' ClusterID set.
func dedupeDraft(drafts []entity.ChainDraft, candidate entity.ChainDraft) []entity.ChainDraft {
	for i := range drafts {
		if sameClusterPath(drafts[i], candidate) {
			return drafts // this path already exists — don't duplicate
		}
	}
	return append(drafts, candidate)
}

func sameClusterPath(left, right entity.ChainDraft) bool {
	if len(left.Participants) != len(right.Participants) {
		return false
	}
	for position := range left.Participants {
		if left.Participants[position].ClusterID != right.Participants[position].ClusterID {
			return false
		}
	}
	return true
}
