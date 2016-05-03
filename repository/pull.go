package repository

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/libgit2/git2go"
)

// Pull a repository branch
func (r *Repository) Pull(branchName string) error {
	origin, err := r.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	defer origin.Free()

	rawLocalBranchRef := fmt.Sprintf("refs/heads/%s", branchName)
	rawRemoteBranchRef := fmt.Sprintf("refs/remotes/origin/%s", branchName)

	// Fetch
	err = origin.Fetch([]string{rawLocalBranchRef}, nil, "")
	if err != nil {
		return err
	}

	// Change the HEAD to current branch and checkout
	err = r.SetHead(rawLocalBranchRef)
	if err != nil {
		return err
	}
	err = r.CheckoutHead(&git.CheckoutOpts{
		Strategy: git.CheckoutForce,
	})
	if err != nil {
		return err
	}

	head, err := r.Head()
	if err != nil {
		return err
	}

	remoteBranchRef, err := r.References.Lookup(rawRemoteBranchRef)
	if err != nil {
		return err
	}

	// If the ref on the branch doesn't exist locally, create it
	// This also creates the branch
	_, err = r.References.Lookup(rawLocalBranchRef)
	if err != nil {
		_, err = r.References.Create(rawLocalBranchRef, remoteBranchRef.Target(), true, "")
		if err != nil {
			return err
		}
	}

	// Create annotated commit
	annotatedCommit, err := r.AnnotatedCommitFromRef(remoteBranchRef)
	if err != nil {
		return err
	}

	// Merge analysis
	mergeHeads := []*git.AnnotatedCommit{annotatedCommit}
	analysis, _, err := r.MergeAnalysis(mergeHeads)
	if err != nil {
		return err
	}

	// Action on analysis
	if analysis&git.MergeAnalysisUpToDate != 0 { // On up-to-date merge
		log.Debugf("Skipping pull on repository %s, branch %s. Already up to date", r.repoConfig.Name, branchName)
	} else if analysis&git.MergeAnalysisFastForward != 0 { // On fast-forward merge
		log.Infof("Changes detected on repository %s branch %s, Fast-forwarding", r.repoConfig.Name, branchName)

		if err := r.Merge(mergeHeads, nil, nil); err != nil {
			return err
		}
	} else if analysis&git.MergeAnalysisNormal != 0 { // On normal merge
		log.Infof("Changes detected on repository %s. Pulling commits from branch %s", r.repoConfig.Name, branchName)

		if err := r.Merge(mergeHeads, nil, nil); err != nil {
			return err
		}
	}

	// Update refs on heads (local) from remotes
	if _, err := head.SetTarget(remoteBranchRef.Target(), ""); err != nil {
		return err
	}

	defer r.StateCleanup()
	return nil
}