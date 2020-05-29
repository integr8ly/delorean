# Prow Commands

Prow commands allow developers to initiate tests and other functionality made available from enabled prow plugins directly from any pull request using comments.

A list of commands available to use on a given repository can be found here "https://deck-ci.svc.ci.openshift.org/command-help?repo=<GitHub Org>%2F<GitHub Repo>". 

## lgtm and approve

Workflow comments that apply labels on the PR when made by appropriate OWNERS, see [Prow Pull Request Workflow](./prow-pr-workflow.md)

Example:
```bash
/lgtm
/approve
```

## cc and uncc

Request review from specific GitHub user (Must exist in the OWNERS file).

Example:
```bash
/cc @mikenairn
```

Remove request.

Example:
```bash
/uncc @mikenairn
```

## hold

Applies a label to the PR "do-not-merge/hold" preventing it from being automatically merged even if someone approves it.

Example:
```bash
This requires extensive testing before being merged.

/hold
```

Hold can be removed using hold cancel

Example:
```bash
Tests have been executed.

/hold cancel
```

## cherrypick

The [Prow Cherrypick plugin](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/cherrypicker), when enabled, automates cherry picking merged PRs back to different branches.

Example:
```bash
/cherrypick v1.5
```

The above comment will result in opening a new PR against the v1.5 branch once the PR where the comment was made gets merged or is already merged.

Note: The cherrypick command requires that the person issuing the command have their membership to the integr8ly GitHub org be public:

![GitHub Membership](./images/github-membership-public.png)

If you don't have public membership, you will get an error like the following:

![Cherrypick Fail](./images/cherrypick-fail-non-public-membership.png)

## Useful Links
 
* [Prow Command Help](https://prow.k8s.io/command-help)
* [Prow Command Help(integreatly-operator)]( https://deck-ci.svc.ci.openshift.org/command-help?repo=integr8ly%2Fintegreatly-operator)
* [Prow Cherrypick plugin](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/cherrypicker)