package cmd

import (
	"context"
	"fmt"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/quay"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"net/http"
	"time"
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
		if err = DoTagRelease(cmd.Context(), ghClient.Git, repoInfo, quayClient, quayRepo, tagReleaseCmdOpts); err != nil {
			handleError(err)
		}
	},
}

func DoTagRelease(ctx context.Context, ghClient services.GitService, gitRepoInfo *githubRepoInfo, quayClient *quay.Client, quayRepo string, cmdOpts *tagReleaseOptions) error {
	rv, err := utils.NewRHMIVersion(cmdOpts.releaseVersion)
	if err != nil {
		return err
	}
	fmt.Println("Fetch git ref:", fmt.Sprintf("refs/heads/%s", cmdOpts.branch))
	headRef, err := getGitRef(ctx, ghClient, gitRepoInfo, fmt.Sprintf("refs/heads/%s", cmdOpts.branch))
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
	fmt.Println("Try to create image tag on quay.io")
	err = tryCreateQuayTag(ctx, quayClient, quayRepo, cmdOpts.branch, rv.TagName(), headRef.GetObject().GetSHA())
	if err != nil {
		if cmdOpts.wait {
			fmt.Println("Wait for the latest image to be available on quay.io. Will check every", cmdOpts.waitInterval, "minutes for", cmdOpts.waitMax, "minutes")
			err = utils.Retry(time.Duration(cmdOpts.waitInterval)*time.Minute, time.Duration(cmdOpts.waitMax)*time.Minute, func() error {
				fmt.Println("Try to create image tag on quay.io")
				err = tryCreateQuayTag(ctx, quayClient, quayRepo, cmdOpts.branch, rv.TagName(), headRef.GetObject().GetSHA())
				if err != nil {
					fmt.Println("Failed. Will try again later.")
				}
				return err
			})
			if err != nil {
				fmt.Println("Can not create image tag on quay.io")
				return err
			}
		} else {
			return err
		}
	}
	fmt.Println("Image tag created:", rv.TagName())
	return nil
}

func getGitRef(ctx context.Context, client services.GitService, gitRepoInfo *githubRepoInfo, ref string) (*github.Reference, error) {
	gitRef, resp, err := client.GetRef(ctx, gitRepoInfo.owner, gitRepoInfo.repo, ref)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return gitRef, nil
}

func createGitTag(ctx context.Context, client services.GitService, gitRepoInfo *githubRepoInfo, tag string, sha string) (*github.Reference, error) {
	tagRefVal := fmt.Sprintf("refs/tags/%s", tag)
	tagRef, err := getGitRef(ctx, client, gitRepoInfo, tagRefVal)
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

func tryCreateQuayTag(ctx context.Context, quayClient *quay.Client, quayRepo string, existingTag string, newTag string, commitSHA string) error {
	tags, _, err := quayClient.Tags.List(ctx, quayRepo, &quay.ListTagsOptions{
		SpecificTag: existingTag,
	})
	if err != nil {
		return err
	}
	if len(tags.Tags) == 0 {
		return fmt.Errorf("tag %s doesn't exit", existingTag)
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
		return fmt.Errorf("can't find an image with given tag %s that matches the given commit SHA: %s", existingTag, commitSHA)
	} else {
		_, err := quayClient.Tags.Change(ctx, quayRepo, newTag, &quay.ChangTag{
			ManifestDigest: *tag.ManifestDigest,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func init() {
	releaseCmd.AddCommand(tagReleaseCmd)
	tagReleaseCmd.Flags().StringVarP(&tagReleaseCmdOpts.branch, "branch", "b", "master", "Branch to create the tag")
	tagReleaseCmd.Flags().BoolVarP(&tagReleaseCmdOpts.wait, "wait", "w", false, "Wait for the quay tag to be created (it could take up to 1 hour)")
	tagReleaseCmd.Flags().Int64Var(&tagReleaseCmdOpts.waitInterval, "wait-interval", 5, "Specify the interval to check tags in quay while waiting. In minutes.")
	tagReleaseCmd.Flags().Int64Var(&tagReleaseCmdOpts.waitMax, "wait-max", 90, "Specify the max wait time for tags be to created in quay. In minutes.")
}
