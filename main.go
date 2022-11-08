package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

const (
	GithubOrg   = "vmware-tanzu"
	CarvelTopic = "carvel"
	PivotalBot  = "pivotal-issuemaster"
	VMwareBot   = "vmwclabot"
)

func main() {
	var (
		beginDate time.Time
		endDate   time.Time
		err       error
	)

	// validate expected environment variables
	github_api_token := os.Getenv("GITHUB_API_TOKEN")
	if github_api_token == "" {
		panic("GITHUB_API_TOKEN is not set")
	}

	if os.Getenv("BEGIN_DATE") == "" {
		panic("BEGIN_DATE is not set")
	} else {
		beginDate, err = time.Parse(time.RFC3339, os.Getenv("BEGIN_DATE"))
		if err != nil {
			fmt.Println("Failed to parse BEGIN_DATE. Please make sure that it is in RFC3339 format (example: 2006-01-02T15:04:05-08:00)")
			panic(err)
		}
	}

	if os.Getenv("END_DATE") == "" {
		panic("END_DATE is not set")
	} else {
		endDate, err = time.Parse(time.RFC3339, os.Getenv("END_DATE"))
		if err != nil {
			fmt.Println("Failed to parse END_DATE. Please make sure that it is in RFC3339 format (example: 2006-01-02T15:04:05-08:00)")
			panic(err)
		}
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: github_api_token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// get list of public orgs
	opt := &github.RepositoryListByOrgOptions{
		Type:        "public",
		ListOptions: github.ListOptions{PerPage: 1000},
	}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), GithubOrg, opt)
	if err != nil {
		panic(err)
	}

	// get carvel projects from org
	carvelReposToFilter := []*github.Repository{}
	for _, repo := range repos {
		for _, topic := range repo.Topics {
			if topic == CarvelTopic {
				carvelReposToFilter = append(carvelReposToFilter, repo)
			}
		}
	}

        // remove non-Carvel repos
	reposToSkip := []string{
		"kubeapps",
		"tanzu-framework",
		"package-for-kpack",
		"package-for-cartographer",
		"package-for-kubeapps",
		"package-for-helm-controller",
		"package-for-kustomize-controller",
		"package-for-kpack-dependencies",
		"package-for-source-controller",
		"package-for-application-toolkit",
	}

	carvelRepos := []*github.Repository{}
	for _, repoToCheck := range carvelReposToFilter {
		found := false
		for _, repoToSkip := range reposToSkip {
			if *repoToCheck.Name == repoToSkip {
				found = true
			}
		}
		if !found {
			carvelRepos = append(carvelRepos, repoToCheck)
		}
	}

	// get PRs within timeframe for all repos
	pullRequestInfos := []PullRequestInfo{}
	for _, carvelRepo := range carvelRepos {
		var complete bool
		opt := &github.PullRequestListOptions{
			State:       "all",
			Sort:        "created",
			Direction:   "asc",
			ListOptions: github.ListOptions{PerPage: 100},
		}

		repoPRs := []*github.PullRequest{}
		for i := 1; !complete; i++ {
			opt.ListOptions.Page = i
			prs, _, err := client.PullRequests.List(context.Background(), GithubOrg, *carvelRepo.Name, opt)
			if err != nil {
				panic(err)
			}
			repoPRs = append(repoPRs, prs...)
			if len(prs) < 100 {
				complete = true
			}
		}

		fmt.Printf("========== %s ==========\n", *carvelRepo.Name)
		fmt.Printf("Total PRs: %v\n", len(repoPRs))

		filteredPRs := filterPRsByWindow(repoPRs, beginDate, endDate)
		fmt.Printf("Filtered PRs: %v\n", len(filteredPRs))

		for _, pr := range filteredPRs {
			comments, _, err := client.Issues.ListComments(context.Background(), GithubOrg, *carvelRepo.Name, *pr.Number, nil)
			if err != nil {
				panic(err)
			}

			timeOfEngagementForComments := findTimeOfFirstCommentNotFromUser(comments, *pr.User.Login)

			reviews, _, err := client.PullRequests.ListReviews(context.Background(), GithubOrg, *carvelRepo.Name, *pr.Number, &github.ListOptions{PerPage: 1000})
			if err != nil {
				panic(err)
			}

			timeOfEngagementForReviews := findTimeOfEngagementForReviews(reviews)

			timesToConsider := []time.Time{timeOfEngagementForComments, timeOfEngagementForReviews}
			if *pr.State == "closed" {
				timesToConsider = append(timesToConsider, *pr.ClosedAt)
			}
			isMerged, _, err := client.PullRequests.IsMerged(context.Background(), GithubOrg, *carvelRepo.Name, *pr.Number)
			if err != nil {
				panic(err)
			}
			if isMerged {
				timesToConsider = append(timesToConsider, *pr.MergedAt)
			}

			// get the earliest time of engagement
			timeOfEngagement := findNonZeroMinimum(timesToConsider)
			dur := timeOfEngagement.Sub(*pr.CreatedAt)
			if timeOfEngagement.IsZero() {
				// if a PR is still open, set the duration to an arbitrarily high value
				// so that it results near the end of the sorted list
				dur, err = time.ParseDuration("10000h")
				if err != nil {
					panic(err)
				}
			}

			pullRequestInfos = append(pullRequestInfos,
				PullRequestInfo{
					Repo:             carvelRepo,
					PR:               pr,
					TimeToEngagement: dur,
				},
			)
		}
	}

	sort.SliceStable(pullRequestInfos, func(i, j int) bool {
		return pullRequestInfos[i].TimeToEngagement < pullRequestInfos[j].TimeToEngagement
	})

	for i, prInfo := range pullRequestInfos {
		fmt.Printf("%v. %v #%v: %v by %v\n", i+1, *prInfo.Repo.Name, *prInfo.PR.Number, prInfo.TimeToEngagement, *prInfo.PR.User.Login)
	}

	fmt.Println("# of PRs: ", len(pullRequestInfos))
	fmt.Println("Median: ", findMedian(pullRequestInfos))
}

type PullRequestInfo struct {
	Repo             *github.Repository
	PR               *github.PullRequest
	TimeToEngagement time.Duration
}

// this function expects that prs is sorted by CreatedAt in an ascending order
func filterPRsByWindow(prs []*github.PullRequest, begin, end time.Time) []*github.PullRequest {
	filteredPRs := []*github.PullRequest{}

	for _, pr := range prs {
		if pr.CreatedAt.Before(begin) {
			continue
		}

		if pr.CreatedAt.After(end) {
			break
		}

		filteredPRs = append(filteredPRs, pr)
	}

	return filteredPRs
}

//  find time of first comment not from vmwclabot or pivotal-issuemaster or requester
func findTimeOfFirstCommentNotFromUser(comments []*github.IssueComment, user string) time.Time {
	for _, comment := range comments {
		if *comment.User.Login == user || *comment.User.Login == PivotalBot || *comment.User.Login == VMwareBot {
			continue
		}
		return *comment.CreatedAt
	}
	return time.Time{}
}

//  find time of first review
func findTimeOfEngagementForReviews(reviews []*github.PullRequestReview) time.Time {
	if len(reviews) > 0 {
		return *reviews[0].SubmittedAt
	}
	return time.Time{}
}

// returns time.Time if only zero-values are provided
func findNonZeroMinimum(times []time.Time) time.Time {
	var min time.Time
	for _, time := range times {
		if time.IsZero() {
			continue
		}

		if min.IsZero() {
			min = time
		}

		if time.Before(min) {
			min = time
		}
	}
	return min
}

func findMedian(prInfos []PullRequestInfo) time.Duration {
	count := len(prInfos)
	if count == 0 {
		return 0
	}

	mid := int(math.Floor(float64(count) / 2.0))
	// if even
	if count%2 == 0 {
		return (prInfos[mid-1].TimeToEngagement + prInfos[mid].TimeToEngagement) / 2
	} else { // odd
		return prInfos[mid].TimeToEngagement
	}
}
