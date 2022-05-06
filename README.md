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
GITHUB_API_TOKEN="notatoken" BEGIN_DATE="2022-01-29T00:00:00-08:00" END_DATE="2022-04-29T23:59:59-08:00" go run main.go
```

# How Aaron collects quarterly metrics
1. export variables:

    ```
    export GITHUB_API_TOKEN="notatoken"
    export BEGIN_DATE="notadate"
    export END_DATE="notadate"
    export REPORT_OUTPUT="fy23q1.out"
    export CONTRIBUTORS_OUTPUT="fy23q1contributors.out"
    ```

1. run the report and save the results:

    `go run main.go > "${REPORT_OUTPUT}"`

1. get contributors:

    `cut -d' ' -f 6 "${REPORT_OUTPUT}"| sort | uniq -c | sort -r > "${CONTRIBUTORS_OUTPUT}"`

1. manually filter contributors by adding `(non-VMW)` to the end of any
   non-VMware contributors
1. process contributor metrics
    1. For total # of contributors: `wc -l "${CONTRIBUTORS_OUTPUT}"`
    1. For total # of non-VMW contributors: `grep non-VMW "${CONTRIBUTORS_OUTPUT}" | wc -l`
    1. For total # of non-VMW contributions (manually add): `grep non-VMW "${CONTRIBUTORS_OUTPUT}"`
	
