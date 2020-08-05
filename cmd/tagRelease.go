package cmd

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/quay"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	commitIDLabelFilter = "io.openshift.build.commit.id"
)

type tagReleaseConfigGithubRepo struct {
	Owner string
	Repo  string
}

type tagReleaseConfigRepo struct {
	Name           string
	SkipPreRelease bool
	GithubRepo     tagReleaseConfigGithubRepo
	QuayRepos      []string
}

type tagReleaseConfig struct {
	Version      string
	Branch       string
	Wait         bool
	WaitInterval int64
	WaitMax      int64
	Repositories []tagReleaseConfigRepo
}

type tagReleaseFlags struct {
	configFile string
}

type tagReleaseCmd struct {
	githubClient services.GitService
	quayClient   *quay.Client
	config       *tagReleaseConfig
}

func newTagReleaseCmd(f *tagReleaseFlags) (*tagReleaseCmd, error) {

	c := &tagReleaseConfig{}
	err := utils.PopulateObjectFromYAML(f.configFile, c)
	if err != nil {
		return nil, err
	}

	githubToken, err := requireValue(GithubTokenKey)
	if err != nil {
		handleError(err)
	}

	quayToken, err := requireValue(QuayTokenKey)
	if err != nil {
		handleError(err)
	}

	githubClient := newGithubClient(githubToken)
	quayClient := newQuayClient(quayToken)

	return &tagReleaseCmd{
		githubClient: githubClient.Git,
		quayClient:   quayClient,
		config:       c,
	}, nil
}

func (cmd *tagReleaseCmd) run(ctx context.Context) error {

	version, err := utils.NewRHMIVersion(cmd.config.Version)
	if err != nil {
		return err
	}

	for _, r := range cmd.config.Repositories {
		err = cmd.runRepository(ctx, version, &r)
		if err != nil {
			return fmt.Errorf("failed to tag repository %s with error: %s", r.Name, err)
		}
	}
	return nil
}

func (cmd *tagReleaseCmd) runRepository(ctx context.Context, v *utils.RHMIVersion, r *tagReleaseConfigRepo) error {

	if r.SkipPreRelease && v.IsPreRelease() {
		fmt.Printf("[%s] skip pre-release version: %s\n", r.Name, v)
		return nil
	}

	branchRefName := fmt.Sprintf("refs/heads/%s", cmd.config.Branch)
	fmt.Printf("[%s] fetch git ref: %s\n", r.Name, branchRefName)
	branchHeadRef, err := getGitRef(ctx, cmd.githubClient, &r.GithubRepo, branchRefName, false)
	if err != nil {
		return err
	}
	if branchHeadRef == nil {
		return fmt.Errorf("can not find git ref: %s", branchHeadRef)
	}

	preReleaseTagRefName := fmt.Sprintf("refs/tags/%s-", v.TagName())
	fmt.Printf("[%s] fetch git ref: %s\n", r.Name, preReleaseTagRefName)
	preReleaseTagRef, err := getGitRef(ctx, cmd.githubClient, &r.GithubRepo, preReleaseTagRefName, true)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] create git tag: %s\n", r.Name, v.TagName())
	tagRef, err := createGitTag(ctx, cmd.githubClient, &r.GithubRepo, v.TagName(), branchHeadRef.GetObject().GetSHA())
	if err != nil {
		return err
	}
	fmt.Printf("[%s] git tag %s created: %s\n", r.Name, v.TagName(), tagRef.GetURL())

	if len(r.QuayRepos) > 0 {
		fmt.Printf("[%s] tags image on quay.io\n", r.Name)
		quayDstTag := v.TagName()
		quaySrcTag := v.ReleaseBranchImageTag()
		commitSHA := branchHeadRef.GetObject().GetSHA()

		//If this is a final release and we have an existing tag (rc tag), promote the existing rc tag to the final release, otherwise continue as normal
		if !v.IsPreRelease() && preReleaseTagRef != nil {
			quaySrcTag = strings.Replace(preReleaseTagRef.GetRef(), "refs/tags/", "", -1)
			commitSHA = preReleaseTagRef.GetObject().GetSHA()
		}

		ok := cmd.tryCreateQuayTag(ctx, r, quaySrcTag, quayDstTag, commitSHA)
		if !ok {
			if cmd.config.Wait {
				fmt.Printf("[%s] wait for the latest image with tag %s to be available on quay.io\n", r.Name, quaySrcTag)
				fmt.Printf("[%s] will check every %d minutes for max %d minutes\n", r.Name, cmd.config.WaitInterval, cmd.config.WaitMax)
				err = wait.Poll(time.Duration(cmd.config.WaitInterval)*time.Minute, time.Duration(cmd.config.WaitMax)*time.Minute, func() (bool, error) {
					ok = cmd.tryCreateQuayTag(ctx, r, quaySrcTag, quayDstTag, commitSHA)
					if !ok {
						fmt.Println("[%s] failed to image on quay.io, try again in %d minutes\n", r.Name, cmd.config.WaitInterval)
					}
					return ok, nil
				})
				if err != nil {
					return fmt.Errorf("failed to tag images %v on quay.io with error: %s", r.QuayRepos, err)
				}
			} else {
				return err
			}
		}
		fmt.Printf("[%s] image tags created: %s\n", r.Name, v.TagName())
	} else {
		fmt.Printf("[%s] skip creating image tags as no quay repos specified\n", r.Name)
	}
	return nil
}

func getGitRef(ctx context.Context, client services.GitService, gitRepoInfo *tagReleaseConfigGithubRepo, ref string, mostRecent bool) (*github.Reference, error) {
	gitRefs, resp, err := client.GetRefs(ctx, gitRepoInfo.Owner, gitRepoInfo.Repo, ref)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	//If mostRecent is true, sort the list of refs and return the most recent one based on it's name v1.1.1-rc3 vs v1.1.1-rc2 etc..
	if mostRecent && len(gitRefs) > 0 {
		sort.Slice(gitRefs, func(i, j int) bool {
			return gitRefs[i].GetRef() > gitRefs[j].GetRef()
		})
		return gitRefs[0], nil
	} else {
		for _, r := range gitRefs {
			if r.GetRef() == ref {
				return r, nil
			}
		}
	}
	return nil, nil
}

func createGitTag(ctx context.Context, client services.GitService, gitRepoInfo *tagReleaseConfigGithubRepo, tag string, sha string) (*github.Reference, error) {
	tagRefVal := fmt.Sprintf("refs/tags/%s", tag)
	tagRef, err := getGitRef(ctx, client, gitRepoInfo, tagRefVal, false)
	if err != nil {
		return nil, err
	}
	if tagRef != nil {
		if tagRef.GetObject().GetSHA() != sha {
			return nil, fmt.Errorf("tag %s is already created but pointing to a different commit. Please delete it first", tag)
		}
		return tagRef, nil
	}
	tagRef = &github.Reference{
		Ref: &tagRefVal,
		Object: &github.GitObject{
			SHA: &sha,
		},
	}
	created, _, err := client.CreateRef(ctx, gitRepoInfo.Owner, gitRepoInfo.Repo, tagRef)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (cmd *tagReleaseCmd) tryCreateQuayTag(ctx context.Context, repository *tagReleaseConfigRepo, quaySrcTag string, quayDstTag string, commitSHA string) bool {
	ok := true
	for _, r := range repository.QuayRepos {
		repo, tag := getImageRepoAndTag(r, quayDstTag)
		err := createTagForImage(ctx, cmd.quayClient, *repo, quaySrcTag, *tag, commitSHA)
		if err != nil {
			ok = false
			fmt.Printf("[%s] failed to create the image tag for %s with error: %s\n", repository.Name, r, err)
		} else {
			fmt.Printf("[%s] image tag '%s' created from tag '%s' with commit '%s' in repo '%s'\n", repository.Name, *tag, quaySrcTag, commitSHA, *repo)
		}
	}
	return ok
}

func createTagForImage(ctx context.Context, quayClient *quay.Client, quayRepo string, quaySrcTag string, quayDstTag string, commitSHA string) error {
	tags, _, err := quayClient.Tags.List(ctx, quayRepo, &quay.ListTagsOptions{
		SpecificTag: quaySrcTag,
	})
	if err != nil {
		return err
	}
	if len(tags.Tags) == 0 {
		return fmt.Errorf("tag %s doesn't exit", quaySrcTag)
	}
	tag := tags.Tags[0]
	commitID, _, err := quayClient.Manifests.ListLabels(ctx, quayRepo, *tag.ManifestDigest, &quay.ListManifestLabelsOptions{Filter: commitIDLabelFilter})
	if err != nil {
		return err
	}
	if len(commitID.Labels) == 0 {
		return fmt.Errorf("label %s doesn't exist", commitIDLabelFilter)
	}
	if *commitID.Labels[0].Value != commitSHA {
		return fmt.Errorf("can't find an image with given tag %s that matches the given commit SHA: %s", quaySrcTag, commitSHA)
	} else {
		_, err = quayClient.Tags.Change(ctx, quayRepo, quayDstTag, &quay.ChangTag{
			ManifestDigest: *tag.ManifestDigest,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func getImageRepoAndTag(s string, defaultTag string) (*string, *string) {
	p := strings.Split(s, ":")
	if len(p) > 1 {
		return &p[0], &p[1]
	}
	return &s, &defaultTag
}

func init() {
	var flags = &tagReleaseFlags{}

	// tagReleaseCmd represents the tagRelease command
	var cmd = &cobra.Command{
		Use:   "tag-release",
		Short: "Tag the given repos and images with the given release version",
		Long: `Change a release tag using the given release version for the HEAD of the given branch.
           Also create the same tag for the image that is built from the same commit`,
		Run: func(cmd *cobra.Command, args []string) {

			c, err := newTagReleaseCmd(flags)
			if err != nil {
				handleError(err)
			}

			err = c.run(cmd.Context())
			if err != nil {
				handleError(err)
			}
		},
	}

	cmd.Flags().StringVar(&flags.configFile, "config-file", "", "Path to the configuration file for this command")
	cmd.MarkFlagRequired("config-file")

	releaseCmd.AddCommand(cmd)
}
