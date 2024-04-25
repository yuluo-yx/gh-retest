## Github Pull Request Retest

You can add this action to your project by adding the following to your project's workflows:

```yml
name: Retest Action on PR Comment

on:
  issue_comment:
    types: [created]

permissions:
  contents: read

jobs:
  retest:
    if: |
      ${{
         github.event.issue.pull_request
         && github.repository == 'owner/repo'
      }}
    name: Retest
    runs-on: ubuntu-22.04
    permissions:
      pull-requests: write
      actions: write
    steps:
      - uses: yuluo-yx/gh-retest@VERSION
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          comment-id: ${{ github.event.comment.id }}
          pr-url: ${{ github.event.issue.pull_request.url }}
```

## License

MIT

