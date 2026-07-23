import requests
import os
import sys

token = os.getenv('GITHUB_TOKEN')
repo = os.getenv('REPOSITORY')
github_output = os.getenv('GITHUB_OUTPUT')
headers = {'Authorization': f'token {token}'}

response = requests.get(f'https://api.github.com/repos/{repo}/releases/latest', headers=headers)
response.raise_for_status()
latest_release = response.json()
latest_release_date = latest_release['created_at']
latest_version = latest_release['name'].removeprefix('v')

prs_since = []
page = 1
while True:
    response = requests.get(
        f'https://api.github.com/repos/{repo}/pulls?state=closed&sort=updated&direction=desc&per_page=100&page={page}',
        headers=headers,
    )
    response.raise_for_status()
    page_prs = response.json()
    if not page_prs:
        break
    merged = [pr for pr in page_prs if pr['merged_at'] is not None and pr['merged_at'] > latest_release_date]
    prs_since.extend(merged)
    if len(merged) < len(page_prs):
        break
    page += 1

has_feature = any('kind/feature' in [label['name'] for label in pr['labels']] for pr in prs_since)

major, minor, patch = latest_version.split('.')
if has_feature:
    next_version = f'{major}.{int(minor) + 1}.0'
    reason = 'minor bump, kind/feature PRs found'
else:
    next_version = f'{major}.{minor}.{int(patch) + 1}'
    reason = f'patch bump, no kind/feature PRs since {latest_version}'

if github_output:
    with open(github_output, 'a') as f:
        f.write(f'version={next_version}\n')
else:
    print(next_version)

print(f'::notice ::Auto-detected version: {next_version} ({reason})', file=sys.stderr)
