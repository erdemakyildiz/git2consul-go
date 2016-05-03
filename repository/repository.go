package repository

import (
	"os"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/cleung2010/go-git2consul/config"
	"github.com/libgit2/git2go"
)

type Repository struct {
	*git.Repository
	repoConfig *config.Repo
	store      string
}

func PollRepos(cfg *config.Config) error {
	for _, repo := range cfg.Repos {
		// Create Repository object for each repo
		store := filepath.Join(cfg.LocalStore, repo.Name)
		raw_repo, err := git.OpenRepository(store)
		if err != nil {
			// If cannot open/find repo, clone it
			log.Debugf("Cannot open repository: %s", err)
		}
		r := &Repository{
			raw_repo,
			repo,
			store,
		}

		// Poll repository by interval, or webhook
		go r.pollRepoByInterval()
		// go r.PollRepoByWebhook()
	}

	return nil
}

func (r *Repository) poll() error {
	if _, err := os.Stat(r.store); err != nil {
		// If there is no repo, create and clone
		if os.IsNotExist(err) {
			log.Infof("Repository %s not cached, cloning to %s", r.repoConfig.Name, r.store)
			err := os.Mkdir(r.store, 0755)
			if err != nil {
				return err
			}

			err = r.Clone()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Pull the repository, all specified branches
		for _, branch := range r.repoConfig.Branches {
			r.Pull(branch)
		}
	}

	return nil
}

func (r *Repository) pollRepoByInterval() {
	hooks := r.repoConfig.Hooks
	interval := time.Second

	// Find polling hook
	for _, h := range hooks {
		if h.Type == "polling" {
			interval = h.Interval
			break
		}
	}

	// If no polling found, don't poll
	if interval == 0 {
		return
	}

	// Initial poll
	err := r.poll()
	if err != nil {
		log.Error(err)
	}

	ticker := time.NewTicker(interval * time.Second)
	for {
		select {
		case <-ticker.C:
			err := r.poll()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (r *Repository) pollRepoByWebhook() {

}