import requests
import os
import sys

token = os.getenv('GITHUB_TOKEN')
repo = os.getenv('REPOSITORY')
headers = {'Authorization': f'token {token}'}

response = requests.get(f'https://api.github.com/repos/{repo}/releases/latest', headers=headers)
response.raise_for_status()
latest_release = response.json()
latest_release_date = latest_release['created_at']
latest_version = latest_release['name']

response = requests.get(
    f'https://api.github.com/repos/{repo}/pulls?state=closed&sort=updated&direction=desc',
    headers=headers,
)
response.raise_for_status()
prs_since = [
    pr for pr in response.json()
    if pr['merged_at'] is not None and pr['merged_at'] > latest_release_date
]

has_feature = any('kind/feature' in [label['name'] for label in pr['labels']] for pr in prs_since)

major, minor, patch = latest_version.split('.')
if has_feature:
    next_version = f'{major}.{int(minor) + 1}.0'
    reason = 'minor bump, kind/feature PRs found'
else:
    next_version = f'{major}.{minor}.{int(patch) + 1}'
    reason = f'patch bump, no kind/feature PRs since {latest_version}'

print(next_version)
print(f'::notice ::Auto-detected version: {next_version} ({reason})', file=sys.stderr)
