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

type tagReleaseOptions struct {
	releaseVersion string
	branch         string
	wait           bool
	waitInterval   int64
	waitMax        int64
	quayRepos      string
	olmType        string
	sourceTag      string
}

var tagReleaseCmdOpts = &tagReleaseOptions{}

// tagReleaseCmd represents the tagRelease command
var tagReleaseCmd = &cobra.Command{
	Use:   "tag-release",
	Short: "Tag the integreatly repo and image with the given release",
	Long: `Change a release tag using the given release version for the HEAD of the given branch.
           Also create the same tag for the image that is built from the same commit`,
	Run: func(cmd *cobra.Command, args []string) {
		var ghToken string
		var quayToken string
		var err error
		if ghToken, err = requireValue(GithubTokenKey); err != nil {
			handleError(err)
		}
		if quayToken, err = requireValue(QuayTokenKey); err != nil {
			handleError(err)
		}
		ghClient := newGithubClient(ghToken)
		quayClient := newQuayClient(quayToken)
		repoInfo := &githubRepoInfo{owner: integreatlyGHOrg, repo: integreatlyOperatorRepo}
		tagReleaseCmdOpts.releaseVersion = releaseVersion
		tagReleaseCmdOpts.olmType = olmType
		if err = DoTagRelease(cmd.Context(), ghClient.Git, repoInfo, quayClient, tagReleaseCmdOpts); err != nil {
			handleError(err)
		}
	},
}

func DoTagRelease(ctx context.Context, ghClient services.GitService, gitRepoInfo *githubRepoInfo, quayClient *quay.Client, cmdOpts *tagReleaseOptions) error {
	rv, err := utils.NewVersion(cmdOpts.releaseVersion, cmdOpts.olmType)
	if err != nil {
		return err
	}
	fmt.Println("Fetch git ref:", fmt.Sprintf("refs/heads/%s", cmdOpts.branch))
	headRef, err := getGitRef(ctx, ghClient, gitRepoInfo, fmt.Sprintf("refs/heads/%s", cmdOpts.branch), false)
	if err != nil {
		return err
	}
	fmt.Println("Fetch git ref:", fmt.Sprintf("refs/tags/%s", rv.TagName()))
	existingRCTagRef, err := getGitRef(ctx, ghClient, gitRepoInfo, fmt.Sprintf("refs/tags/%s", rv.RCTagRef()), true)
	if err != nil {
		return err
	}
	fmt.Println("Create git tag:", rv.TagName())
	if headRef == nil {
		return fmt.Errorf("can not find git ref: refs/heads/%s", cmdOpts.branch)
	}
	tagRef, err := createGitTag(ctx, ghClient, gitRepoInfo, rv.TagName(), headRef.GetObject().GetSHA())
	if err != nil {
		return err
	}
	fmt.Println("Git tag", rv.TagName(), "created:", tagRef.GetURL())
	if len(cmdOpts.quayRepos) > 0 {
		var quaySrcTag string
		fmt.Println("Try to create image tags on quay.io:")
		quayRepos := cmdOpts.quayRepos
		quayDstTag := rv.TagName()
		quaySrcTag = rv.ReleaseBranchImageTag()
		//If this is an OSDe2e image, and destination tag has been passed through the pipeline, set this tag, otherwise continue as normal
		if len(cmdOpts.sourceTag) > 0 {
			quaySrcTag = cmdOpts.sourceTag
		}
		commitSHA := headRef.GetObject().GetSHA()

		//If this is a final release and we have an existing tag (rc tag), promote the existing rc tag to the final release, otherwise continue as normal
		if !rv.IsPreRelease() && existingRCTagRef != nil {
			quaySrcTag = strings.Replace(existingRCTagRef.GetRef(), "refs/tags/", "", -1)
			commitSHA = existingRCTagRef.GetObject().GetSHA()
		}

		ok := tryCreateQuayTag(ctx, quayClient, quayRepos, quaySrcTag, quayDstTag, commitSHA)
		if !ok {
			if cmdOpts.wait {
				fmt.Println("Wait for the latest image to be available on quay.io. Will check every", cmdOpts.waitInterval, "minutes for", cmdOpts.waitMax, "minutes")
				err = wait.Poll(time.Duration(cmdOpts.waitInterval)*time.Minute, time.Duration(cmdOpts.waitMax)*time.Minute, func() (bool, error) {
					ok = tryCreateQuayTag(ctx, quayClient, quayRepos, quaySrcTag, quayDstTag, commitSHA)
					if !ok {
						fmt.Println("Failed. Will try again later.")
					}
					return ok, nil
				})
				if err != nil {
					fmt.Println("Can not create image tag on quay.io")
					return err
				}
			} else {
				return err
			}
		}
		fmt.Println("Image tags created:", rv.TagName())
	} else {
		fmt.Println("Skip creating image tags as no quay repos specified")
	}
	return nil
}

func getGitRef(ctx context.Context, client services.GitService, gitRepoInfo *githubRepoInfo, ref string, mostRecent bool) (*github.Reference, error) {
	gitRefs, resp, err := client.GetRefs(ctx, gitRepoInfo.owner, gitRepoInfo.repo, ref)
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

func createGitTag(ctx context.Context, client services.GitService, gitRepoInfo *githubRepoInfo, tag string, sha string) (*github.Reference, error) {
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
	created, _, err := client.CreateRef(ctx, gitRepoInfo.owner, gitRepoInfo.repo, tagRef)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func tryCreateQuayTag(ctx context.Context, quayClient *quay.Client, quayRepos string, quaySrcTag string, quayDstTag string, commitSHA string) bool {
	repos := strings.Split(quayRepos, ",")
	ok := true
	for _, r := range repos {
		repo, tag := getImageRepoAndTag(r, quayDstTag)
		err := createTagForImage(ctx, quayClient, *repo, quaySrcTag, *tag, commitSHA)
		if err != nil {
			ok = false
			fmt.Println("Failed to create the image tag for", r, "due to error:", err)
		} else {
			fmt.Println(fmt.Sprintf("Image tag '%s' created from tag '%s' with commit '%s' in repo '%s'", *tag, quaySrcTag, commitSHA, *repo))
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
	releaseCmd.AddCommand(tagReleaseCmd)
	tagReleaseCmd.Flags().StringVarP(&tagReleaseCmdOpts.branch, "branch", "b", "master", "Branch to create the tag")
	tagReleaseCmd.Flags().StringVar(&tagReleaseCmdOpts.quayRepos, "quayRepos", fmt.Sprintf("%s,%s", DefaultIntegreatlyOperatorQuayRepo, DefaultIntegreatlyOperatorTestQuayRepo), "Quay repositories. Multiple repos can be specified and separated by ','")
	tagReleaseCmd.Flags().BoolVarP(&tagReleaseCmdOpts.wait, "wait", "w", false, "Wait for the quay tag to be created (it could take up to 1 hour)")
	tagReleaseCmd.Flags().Int64Var(&tagReleaseCmdOpts.waitInterval, "wait-interval", 5, "Specify the interval to check tags in quay while waiting. In minutes.")
	tagReleaseCmd.Flags().Int64Var(&tagReleaseCmdOpts.waitMax, "wait-max", 90, "Specify the max wait time for tags be to created in quay. In minutes.")
	tagReleaseCmd.Flags().StringVar(&tagReleaseCmdOpts.sourceTag, "sourceTag", "", "OSD Source Tag passed through pipeline.")
}
