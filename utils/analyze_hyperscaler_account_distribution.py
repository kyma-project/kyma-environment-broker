#!/usr/bin/env python3
"""
Script to analyze hyperscaler account distribution per global account and per provider.
"""

import subprocess
import json
import sys
from collections import defaultdict
from typing import Dict, List
import argparse


class HyperscalerAccountAnalyzer:
    """Analyzes hyperscaler account distribution per global account and per provider."""
    
    PROVIDERS = ['aws', 'gcp', 'azure', 'alicloud', 'openstack', 'other']
    
    def __init__(self):
        self.hyperscaler_account_count = defaultdict(int)
        self.global_account_data = defaultdict(lambda: defaultdict(list))
        
    def fetch_runtime_data(self) -> List[Dict]:
        try:
            result = subprocess.run(
                ['kcp', 'rt', '-ojson'],
                capture_output=True,
                text=True,
                check=True
            )
            data = json.loads(result.stdout)
            return data.get('data', [])
        except subprocess.CalledProcessError as e:
            print(f"Error executing kcp command: {e}", file=sys.stderr)
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"Error parsing JSON: {e}", file=sys.stderr)
            sys.exit(1)
        except FileNotFoundError:
            print("Error: 'kcp' command not found. Please ensure it's installed and in your PATH.", file=sys.stderr)
            sys.exit(1)
    
    def determine_provider(self, secret_name: str) -> str:
        """Determine cloud provider from hyperscaler account secret name."""
        if not secret_name:
            return None
        
        secret_lower = secret_name.lower()
        
        if 'aws' in secret_lower:
            return 'aws'
        elif 'gcp' in secret_lower:
            return 'gcp'
        elif 'azure' in secret_lower or 'sap-skr' in secret_lower:
            return 'azure'
        elif 'ali' in secret_lower:
            return 'alicloud'
        elif 'openstack' in secret_lower:
            return 'openstack'
        else:
            return 'other'
    
    def analyze(self, data: List[Dict]) -> None:
        """Analyze runtime data and count unique hyperscaler accounts per global account per provider."""
        # Track unique hyperscaler accounts per global account to avoid double counting
        seen_hyperscaler_accounts = defaultdict(set)
        
        for runtime in data:
            global_account_id = runtime.get('globalAccountID')
            secret_name = runtime.get('subscriptionSecretName')
            
            # Skip if no secret name (hyperscaler account identifier)
            if not secret_name:
                continue
            
            provider = self.determine_provider(secret_name)
            if not provider:
                continue
            
            # Use "unknown" if no global account ID
            if not global_account_id:
                global_account_id = "unknown"
            
            # Create unique key for this hyperscaler account in this global account
            account_key = f"{global_account_id}|{secret_name}"
            
            # Skip if we've already counted this hyperscaler account for this global account
            if account_key in seen_hyperscaler_accounts[provider]:
                continue
            
            # Mark this hyperscaler account as seen
            seen_hyperscaler_accounts[provider].add(account_key)
            
            # Collect statistics
            self.hyperscaler_account_count[provider] += 1
            self.global_account_data[global_account_id][provider].append(secret_name)
    
    def get_statistics(self) -> Dict:
        """Calculate hyperscaler account statistics per provider."""
        total = sum(self.hyperscaler_account_count.values())
        
        stats = {
            'total': total,
            'providers': dict(self.hyperscaler_account_count),
            'percentages': {},
            'global_account_count': len(self.global_account_data)
        }
        
        if total > 0:
            for provider in self.PROVIDERS:
                count = self.hyperscaler_account_count.get(provider, 0)
                stats['percentages'][provider] = round((count / total) * 100, 2)
        
        return stats
    
    def get_global_account_breakdown(self) -> Dict:
        """Get breakdown of hyperscaler accounts per global account per provider."""
        breakdown = {}
        
        for ga_id, providers in self.global_account_data.items():
            breakdown[ga_id] = {
                'providers': {},
                'hyperscaler_accounts': {}
            }
            
            for provider in self.PROVIDERS:
                accounts = providers.get(provider, [])
                breakdown[ga_id]['providers'][provider] = len(accounts)
                breakdown[ga_id]['hyperscaler_accounts'][provider] = accounts
        
        return breakdown
    
    def print_table_output(self) -> None:
        """Print hyperscaler account analysis in table format."""
        stats = self.get_statistics()
        breakdown = self.get_global_account_breakdown()
        
        print("\n" + "=" * 70)
        print("Hyperscaler Account Distribution Analysis")
        print("=" * 70)
        print("\n--- Overall Statistics ---\n")
        print(f"{'Provider':<15} {'Accounts':<10} {'Percentage':<12}")
        print("-" * 40)
        
        for provider in self.PROVIDERS:
            count = stats['providers'].get(provider, 0)
            percentage = stats['percentages'].get(provider, 0)
            if count > 0:
                print(f"{provider.upper():<15} {count:<10} {percentage:<12.2f}%")
        
        print("-" * 40)
        print(f"{'TOTAL':<15} {stats['total']:<10}")
        print(f"\nTotal Global Accounts: {stats['global_account_count']}")
        
        print("\n--- Distribution by Global Account ---\n")
        print(f"{'Global Account ID':<38} | {'AWS':<4} | {'GCP':<4} | {'Azure':<5} | {'AliCloud':<8} | {'OpenStack':<10} | {'Other':<5} | {'Total':<5}")
        print("-" * 130)
        
        for ga_id in sorted(breakdown.keys()):
            data = breakdown[ga_id]
            aws_count = data['providers'].get('aws', 0)
            gcp_count = data['providers'].get('gcp', 0)
            azure_count = data['providers'].get('azure', 0)
            alicloud_count = data['providers'].get('alicloud', 0)
            openstack_count = data['providers'].get('openstack', 0)
            other_count = data['providers'].get('other', 0)
            total_count = sum(data['providers'].values())
            
            print(f"{ga_id:<38} | {aws_count:<4} | {gcp_count:<4} | {azure_count:<5} | {alicloud_count:<8} | {openstack_count:<10} | {other_count:<5} | {total_count:<5}")
        
        # Detect global accounts with more than 1 hyperscaler account per provider
        multi_account_gas = []
        for ga_id in sorted(breakdown.keys()):
            data = breakdown[ga_id]
            aws_count = data['providers'].get('aws', 0)
            gcp_count = data['providers'].get('gcp', 0)
            azure_count = data['providers'].get('azure', 0)
            alicloud_count = data['providers'].get('alicloud', 0)
            openstack_count = data['providers'].get('openstack', 0)
            
            if aws_count > 1 or gcp_count > 1 or azure_count > 1 or alicloud_count > 1 or openstack_count > 1:
                multi_account_gas.append({
                    'ga_id': ga_id,
                    'aws': aws_count,
                    'gcp': gcp_count,
                    'azure': azure_count,
                    'alicloud': alicloud_count,
                    'openstack': openstack_count
                })
        
        if multi_account_gas:
            print("\n--- Global Accounts with Multiple Hyperscaler Accounts per Provider ---\n")
            print(f"{'Global Account ID':<38} | {'AWS':<4} | {'GCP':<4} | {'Azure':<5} | {'AliCloud':<8} | {'OpenStack':<10}")
            print("-" * 110)
            for account in multi_account_gas:
                print(f"{account['ga_id']:<38} | {account['aws']:<4} | {account['gcp']:<4} | {account['azure']:<5} | {account['alicloud']:<8} | {account['openstack']:<10}")
            print(f"\nTotal Global Accounts using multiple hyperscaler accounts per provider: {len(multi_account_gas)}")
        
        print("\n" + "=" * 70)
    
    def print_json_output(self) -> None:
        stats = self.get_statistics()
        breakdown = self.get_global_account_breakdown()
        
        output = {
            'statistics': stats,
            'global_accounts': breakdown
        }
        
        print(json.dumps(output, indent=2))


def main():
    parser = argparse.ArgumentParser(
        description='Analyze hyperscaler account distribution per global account and per provider'
    )
    parser.add_argument(
        '--output', '-o',
        choices=['table', 'json'],
        default='table',
        help='Output format (default: table)'
    )
    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='Show detailed hyperscaler account names'
    )
    parser.add_argument(
        '--show-other',
        action='store_true',
        help='Show detailed breakdown of "other" category hyperscaler accounts'
    )
    
    args = parser.parse_args()
    
    analyzer = HyperscalerAccountAnalyzer()
    
    print("Fetching runtime data...", file=sys.stderr)
    data = analyzer.fetch_runtime_data()
    
    print("Analyzing data...", file=sys.stderr)
    analyzer.analyze(data)
    
    if args.output == 'json':
        analyzer.print_json_output()
    else:
        analyzer.print_table_output()
        
        if args.show_other:
            print("\n--- Detailed 'Other' Category Breakdown ---\n")
            breakdown = analyzer.get_global_account_breakdown()
            other_accounts = []
            for ga_id, data in breakdown.items():
                for account in data['hyperscaler_accounts'].get('other', []):
                    other_accounts.append((ga_id, account))
            
            if other_accounts:
                print(f"Total 'Other' hyperscaler accounts: {len(other_accounts)}\n")
                print(f"{'Global Account ID':<38} | Hyperscaler Account Name")
                print("-" * 120)
                for ga_id, account in sorted(other_accounts, key=lambda x: x[1]):
                    print(f"{ga_id:<38} | {account}")
            else:
                print("No hyperscaler accounts in 'Other' category")
        
        if args.verbose:
            print("\n--- Detailed Hyperscaler Account Names by Global Account ---\n")
            breakdown = analyzer.get_global_account_breakdown()
            for ga_id in sorted(breakdown.keys()):
                print(f"\nGlobal Account: {ga_id}")
                data = breakdown[ga_id]
                for provider in analyzer.PROVIDERS:
                    accounts = data['hyperscaler_accounts'].get(provider, [])
                    if accounts:
                        print(f"  {provider.upper()}:")
                        for account in accounts:
                            print(f"    - {account}")


if __name__ == '__main__':
    main()
