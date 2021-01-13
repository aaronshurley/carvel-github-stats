# github-stats
This is a hacky solution to get some stats about [Carvel](https://carvel.dev)
repos. At this time, it returns the following information:
- by repo
  - Number of Total PRs
  - Number of Filtered PRs (within the provided timeframe)
- aggregate
  - Number of Filtered PRs (within the provided timeframe)
  - Sorted list of PRs by Time To Engagement (ascending order)
  - Median Time To Engagement for Filtered PRs

# To Run
1. clone the repo
1. set the required environment variables: `GITHUB_API_TOKEN`, `BEGIN_DATE`, `END_DATE`
  1. to generate a GitHub API Token, follow [these instructions](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token)
1. run

Example:
```
GITHUB_API_TOKEN="notatoken" BEGIN_DATE="2020-10-01T00:00:00-08:00" END_DATE="2020-10-30T23:59:59-08:00" go run main.go
```
