#!/usr/bin/env python3
import re
import yaml
import io

# This script generates Markdown documentation for environment variables defined in a Helm deployment YAML and values.yaml.
# It extracts environment variables from the deployment template, maps them to their descriptions and default values from values.yaml (including comments),
# and outputs a Markdown table. It handles static, single, and composite value references robustly.


# Custom loader to load all scalars as strings (including booleans)
class StrLoader(yaml.SafeLoader):
    pass

def str_constructor(loader, node):
    return loader.construct_scalar(node)

# Remove boolean resolver so 'true'/'false' are loaded as strings
for bool_tag in ['bool', 'bool#yes', 'bool#no']:
    if bool_tag in yaml.SafeLoader.yaml_implicit_resolvers:
        del yaml.SafeLoader.yaml_implicit_resolvers[bool_tag]
StrLoader.add_constructor('tag:yaml.org,2002:bool', str_constructor)
StrLoader.add_constructor('tag:yaml.org,2002:str', str_constructor)

# --- JOB PATHS AND DOCS ---
SINGLE_JOBS = [
    ("resources/keb/templates/deployment.yaml", "docs/contributor/02-30-keb-configuration.md"),
    ("resources/keb/templates/deprovision-retrigger-job.yaml", "docs/contributor/06-50-deprovision-retrigger-cronjob.md"),
    ("utils/archiver/kyma-environment-broker-archiver.yaml", "docs/contributor/06-60-archiver-job.md"),
    ("resources/keb/templates/service-binding-cleanup-job.yaml", "docs/contributor/06-70-service-binding-cleanup-cronjob.md"),
    ("resources/keb/templates/runtime-reconciler-deployment.yaml", "docs/contributor/07-10-runtime-reconciler.md"),
    ("resources/keb/templates/subaccount-sync-deployment.yaml", "docs/contributor/07-20-subaccount-sync.md"),
    ("resources/keb/templates/migrator-job.yaml", "docs/contributor/07-30-schema-migrator.md"),
]
MULTI_JOBS_IN_ONE_TEMPLATE = [
    ("resources/keb/templates/subaccount-cleanup-job.yaml", "docs/contributor/06-30-subaccount-cleanup-cronjob.md", "Subaccount Cleanup CronJob")
]
COMBINED_JOBS_IN_ONE_MD = [
    ("resources/keb/templates/trial-cleanup-job.yaml", "resources/keb/templates/free-cleanup-job.yaml", "docs/contributor/06-40-trial-free-cleanup-cronjobs.md", "Trial Cleanup CronJob", "Free Cleanup CronJob")
]
VALUES_YAML = "resources/keb/values.yaml"

def extract_env_vars_with_paths(deployment_yaml_path):
    """
    Extract environment variables and their value sources from a Helm deployment YAML.
    Handles static values, single .Values references, and composite values (multiple .Values references in one value).
    Returns a list of tuples: (env_var_name, value_path_or_literal)
    """
    env_vars = []
    with open(deployment_yaml_path, "r") as f:
        lines = f.readlines()
    in_env = False
    current_env = None
    for i, line in enumerate(lines):
        if re.match(r"\s*env:\s*$", line):
            in_env = True
            continue
        if in_env:
            m = re.match(r"\s*-\s*name:\s*([A-Z0-9_]+)", line)
            if m:
                current_env = m.group(1)
                # Look ahead for value line
                for j in range(i+1, min(i+2, len(lines))):
                    val_line = lines[j]
                    # Check for composite value with multiple .Values references
                    if re.search(r'{{.*\.Values\..*}}.*{{.*\.Values\..*}}', val_line):
                        # Extract the whole value string inside 'value:'
                        mval = re.search(r'value:\s*"?(.+?)"?$', val_line)
                        if mval:
                            env_vars.append((current_env, mval.group(1).strip()))
                            break
                    else:
                        # Match any Helm template, including filters
                        mval = re.search(r'value:\s*"?({{.*?}})"?', val_line)
                        if mval:
                            env_vars.append((current_env, mval.group(1).strip()))
                            break
                        elif 'valueFrom:' in val_line:
                            env_vars.append((current_env, None))
                            break
                else:
                    env_vars.append((current_env, None))
            elif re.match(r"\s*-\s*name:", line):
                continue
            elif re.match(r"\s*ports:\s*", line):
                in_env = False
    return env_vars

def extract_env_vars_with_paths_multiple_jobs(deployment_yaml_path):
    """
    Extract environment variables for each job in a multi-job YAML file (split by ---).
    Returns a list of lists: [[(env_var_name, value_path_or_literal), ...], ...] for each job.
    Improved: Extracts full Helm value template for each env var, just like main deployment extraction.
    """
    with open(deployment_yaml_path, "r") as f:
        content = f.read()
    job_yamls = content.split('---')
    all_env_vars = []
    for job_yaml in job_yamls:
        lines = job_yaml.splitlines()
        env_vars = []
        in_env = False
        current_env = None
        for i, line in enumerate(lines):
            if re.match(r"\s*env:\s*$", line):
                in_env = True
                continue
            if in_env:
                m = re.match(r"\s*-\s*name:\s*([A-Z0-9_]+)", line)
                if m:
                    current_env = m.group(1)
                    # Look ahead for value line
                    for j in range(i+1, min(i+3, len(lines))):
                        val_line = lines[j]
                        # Extract full Helm value template if present
                        mval = re.search(r'value:\s*"?({{.*?}})"?', val_line)
                        if mval:
                            env_vars.append((current_env, mval.group(1).strip()))
                            break
                        # Composite value: multiple .Values in one line
                        elif re.search(r'{{.*\\.Values\\..*}}.*{{.*\\.Values\\..*}}', val_line):
                            mval2 = re.search(r'value:\s*"?(.+?)"?$', val_line)
                            if mval2:
                                env_vars.append((current_env, mval2.group(1).strip()))
                                break
                        elif 'valueFrom:' in val_line:
                            env_vars.append((current_env, None))
                            break
                    else:
                        env_vars.append((current_env, None))
                elif re.match(r"\s*-\s*name:", line):
                    continue
                elif re.match(r"\s*ports:\s*", line):
                    in_env = False
        if env_vars:
            all_env_vars.append(env_vars)
    return all_env_vars

def parse_values_yaml_with_comments(values_yaml_path):
    """
    Parse values.yaml, extracting a mapping from dot-paths to (description, default value).
    Comments immediately preceding a key are used as the description for that key.
    Returns a dict: {dot.path: {description: str, default: value}}
    """
    with open(values_yaml_path, "r") as f:
        lines = f.readlines()
    # Use custom loader to keep all values as strings
    yaml_data = yaml.load(open(values_yaml_path), Loader=StrLoader)
    doc_map = {}
    path_stack = []
    comment_accumulator = []
    for idx, line in enumerate(lines):
        # Comments: accumulate until next key
        if line.strip().startswith('#'):
            comment_accumulator.append(line.strip('# ').strip())
            continue
        # Key
        m = re.match(r'^(\s*)([a-zA-Z0-9_\-]+):', line)
        if m:
            indent, key = len(m.group(1)), m.group(2)
            # Maintain stack for nested keys
            while path_stack and path_stack[-1][0] >= indent:
                path_stack.pop()
            path_stack.append((indent, key))
            # Compose full key path
            full_key = ".".join([k for _, k in path_stack])
            # Get value from yaml_data
            try:
                val = yaml_data
                for k in full_key.split('.'):
                    val = val[k]
                # If value is a dict, skip (not a leaf)
                if isinstance(val, dict):
                    value = ''
                else:
                    value = val
            except Exception:
                value = ''
            doc_map[full_key] = {
                'description': ' '.join(comment_accumulator).strip(),
                'default': value
            }
            comment_accumulator = []  # Reset after assigning to a key
        # If not a key or comment, do not reset accumulator
    return doc_map

def normalize_path(path):
    """
    Normalize a YAML path for matching: replace _ and - with . and lowercase.
    Does not split camelCase.
    """
    if not path:
        return ''
    return path.replace('_', '.').replace('-', '.').lower()

def extract_helm_value_path(value):
    """
    Extracts the .Values path from a Helm template value string.
    E.g., '{{ .Values.cis.v2.eventServiceURL | required "..." | quote }}' -> 'cis.v2.eventServiceURL'
    Returns None if not found.
    """
    if not value:
        return None
    # Match .Values.<path> optionally followed by filters (| ...)
    m = re.search(r'\{\{\s*\.Values\.([a-zA-Z0-9_.]+)', value)
    if m:
        return m.group(1)
    # Also match if value is just .Values.<path> (no curly braces)
    m2 = re.match(r'\.Values\.([a-zA-Z0-9_.]+)', value)
    if m2:
        return m2.group(1)
    return None

def map_env_to_values(env_vars, values_doc):
    """
    Map extracted environment variables to their descriptions and default values from values.yaml.
    Handles:
      - Static values (not mapped to .Values): description and value are '-'.
      - Single .Values references: use description and default from values.yaml if available.
      - Composite values (multiple .Values references): join descriptions and defaults from all referenced keys.
    Returns a list of dicts: {env, description, default}
    """
    result = []
    norm_doc_map = {normalize_path(k): v for k, v in values_doc.items()}
    for env, path in env_vars:
        desc = ''
        default = ''
        path = path.strip() if path else ''
        doc_entry = None
        norm_path = normalize_path(path) if path else ''
        # Handle composite values like '{{ .Values.host }}.{{ .Values.global.ingress.domainName }}'
        if path and re.search(r'\{\{\s*\.Values\.', path) and '}}.{{' in path:
            # Extract all .Values paths in the composite
            parts = re.findall(r'\.Values\.([a-zA-Z0-9_.]+)', path)
            descs = []
            defaults = []
            for part in parts:
                doc = values_doc.get(part) or norm_doc_map.get(normalize_path(part))
                if doc:
                    if doc.get('description', ''):
                        descs.append(doc['description'])
                    if doc.get('default', ''):
                        defaults.append(str(doc['default']))
            desc = ' / '.join(descs) if descs else '-'
            default = '.'.join(defaults) if defaults else '-'
        else:
            # Try to extract .Values.<path> from any Helm template in the value
            helm_path = extract_helm_value_path(path)
            if helm_path:
                doc_entry = values_doc.get(helm_path) or norm_doc_map.get(normalize_path(helm_path))
                if doc_entry:
                    desc = doc_entry.get('description', '')
                    default = doc_entry.get('default', '')
            elif path and path in values_doc:
                doc_entry = values_doc[path]
                if doc_entry:
                    desc = doc_entry.get('description', '')
                    default = doc_entry.get('default', '')
            elif norm_path and norm_path in norm_doc_map:
                doc_entry = norm_doc_map[norm_path]
                if doc_entry:
                    desc = doc_entry.get('description', '')
                    default = doc_entry.get('default', '')
        result.append({
            'env': env,
            'description': desc,
            'default': default
        })
    return result

def soft_break(text, max_len, prefer_char=None):
    """
    Insert a soft break (\u200b) into text at max_len intervals.
    If prefer_char is set, break at the nearest prefer_char before max_len, otherwise break at max_len.
    """
    if not text or len(text) <= max_len:
        return text
    result = ''
    start = 0
    while start < len(text):
        if len(text) - start <= max_len:
            result += text[start:]
            break
        if prefer_char:
            chunk = text[start:start+max_len]
            last = chunk.rfind(prefer_char)
            if last == -1:
                # No prefer_char, just break at max_len
                result += text[start:start+max_len] + '&#x200b;'
                start += max_len
            else:
                result += text[start:start+last+1] + '&#x200b;'
                start += last+1
        else:
            result += text[start:start+max_len] + '&#x200b;'
            start += max_len
    return result

def parse_existing_table(md_path):
    """
    Parse the first Markdown table in the file. Returns a dict mapping env var (no formatting) to {'default': val, 'description': desc}.
    """
    table = {}
    with open(md_path, 'r') as f:
        lines = f.readlines()
    in_table = False
    for line in lines:
        if line.strip().startswith('| Environment variable') or line.strip().startswith('| Environment Variable'):
            in_table = True
            continue
        if in_table:
            if not line.strip().startswith('|') or line.strip().startswith('|-'):
                if line.strip() == '' or line.strip().startswith('|-'):
                    continue
                else:
                    break
            parts = [p.strip() for p in line.strip().strip('|').split('|')]
            if len(parts) < 3:
                continue
            env = parts[0]
            # Remove formatting
            env_clean = env.replace('**', '').replace('&#x200b;', '').replace('<br>', '').replace('<br/>', '').replace('<br />', '').replace('\u200b', '').replace('\u200B', '').replace(' ', '').replace('\n', '').replace('\r', '')
            default = parts[1]
            desc = parts[2]
            table[env_clean] = {'default': default, 'description': desc}
    return table

def extract_table_markdown(env_docs, existing_table=None):
    buf = io.StringIO()
    buf.write("| Environment Variable | Current Value | Description |\n")
    buf.write("|---------------------|------------------------------|---------------------------------------------------------------|\n")
    for doc in env_docs:
        desc = doc['description'] if doc['description'] else '-'
        default_val = doc['default']
        env_val = soft_break(doc["env"], 20, prefer_char='_')
        env_col = f'**{env_val}**'
        env_key = doc["env"].replace('**', '').replace('&#x200b;', '').replace('<br>', '').replace('<br/>', '').replace('<br />', '').replace('\u200b', '').replace('\u200B', '').replace(' ', '').replace('\n', '').replace('\r', '')
        # Use existing value if new is None or '-'
        if existing_table and env_key in existing_table:
            if default_val is None or str(default_val).strip() == '' or str(default_val).strip() == '-':
                val_col = existing_table[env_key]['default']
            else:
                val_col = f'<code>{str(default_val)}</code>'
            if desc is None or desc.strip() == '' or desc.strip() == '-':
                desc = existing_table[env_key]['description']
        else:
            if default_val is None or str(default_val).strip() == '':
                val_col = 'None'
            else:
                val_col = f'<code>{str(default_val)}</code>'
        buf.write(f"| {env_col} | {val_col} | {desc} |\n")
    return buf.getvalue()

def replace_env_table_in_md(md_path, new_table, section_header=None, all_section_headers=None):
    """
    Replace the environment variable table in the Markdown file.
    If section_header is provided, remove all content from the first matching section header in all_section_headers up to the next unrelated section header (or EOF), then insert new_table.
    If not provided, replace the first Markdown table found (remove the old table before inserting the new one).
    """
    with open(md_path, 'r') as f:
        lines = f.readlines()
    out = []
    if section_header and all_section_headers:
        found_section = False
        idx = 0
        # Find the first occurrence of any section header in all_section_headers
        while idx < len(lines):
            if any(lines[idx].strip().startswith(h) for h in all_section_headers):
                found_section = True
                break
            out.append(lines[idx])
            idx += 1
        if found_section:
            # Skip all lines until a section header not in all_section_headers or EOF
            while idx < len(lines):
                if lines[idx].strip().startswith('### '):
                    if not any(lines[idx].strip().startswith(h) for h in all_section_headers):
                        break
                idx += 1
            # Insert the new table(s)
            out.append(new_table)
            # Append the rest of the file
            out.extend(lines[idx:])
        else:
            out.append('\n' + section_header + '\n' + new_table + '\n')
    else:
        # Remove the old table before inserting the new one
        in_table = False
        table_started = False
        idx = 0
        while idx < len(lines):
            line = lines[idx]
            if line.strip().startswith('| Environment variable') or line.strip().startswith('| Environment Variable'):
                # Found the start of the table
                in_table = True
                table_started = True
                out.append(new_table)
                # Skip all lines that are part of the table
                idx += 1
                while idx < len(lines):
                    if lines[idx].strip().startswith('|') or lines[idx].strip() == '':
                        idx += 1
                    else:
                        break
                continue
            if not in_table:
                out.append(line)
            idx += 1
        if not table_started:
            # If no table found, append at end
            out.append('\n' + new_table + '\n')
    with open(md_path, 'w') as f:
        f.writelines(out)

def main():
    """
    Main entry point: extract env vars, map to values.yaml, and write Markdown documentation for all jobs.
    """
    values_doc = parse_values_yaml_with_comments(VALUES_YAML)

    # 1. Single YAMLs in one template file
    for yaml_path, md_path in SINGLE_JOBS:
        env_vars = extract_env_vars_with_paths(yaml_path)
        env_docs = map_env_to_values(env_vars, values_doc)
        existing_table = parse_existing_table(md_path)
        table = extract_table_markdown(env_docs, existing_table)
        replace_env_table_in_md(md_path, table)
        print(f"Markdown documentation table replaced in {md_path}")

    # 2. Multiple YAMLs in one template file
    for yaml_path, md_path, section_title in MULTI_JOBS_IN_ONE_TEMPLATE:
        env_vars_list = extract_env_vars_with_paths_multiple_jobs(yaml_path)
        tables = []
        all_section_headers = []
        for idx, env_vars in enumerate(env_vars_list):
            env_docs = map_env_to_values(env_vars, values_doc)
            existing_table = parse_existing_table(md_path)
            version = f"v{idx+1}"
            section_header = f"### {section_title} {version}"
            all_section_headers.append(section_header)
            tables.append(section_header + "\n\n" + extract_table_markdown(env_docs, existing_table))
        combined = "\n\n".join(tables) + "\n"
        replace_env_table_in_md(md_path, combined, section_header=f"### {section_title} v1", all_section_headers=all_section_headers)
        print(f"{section_title} env documentation updated in {md_path}")

    # 3. Combined jobs in one Markdown file
    for job in COMBINED_JOBS_IN_ONE_MD:
        yaml1, yaml2, md_path, title1, title2 = job
        env_vars1 = extract_env_vars_with_paths(yaml1)
        env_docs1 = map_env_to_values(env_vars1, values_doc)
        env_vars2 = extract_env_vars_with_paths(yaml2)
        env_docs2 = map_env_to_values(env_vars2, values_doc)
        existing_table = parse_existing_table(md_path)
        table1 = extract_table_markdown(env_docs1, existing_table)
        table2 = extract_table_markdown(env_docs2, existing_table)
        combined = f"### {title1}\n\n" + table1 + f"\n\n### {title2}\n\n" + table2 + "\n"
        all_section_headers = [f"### {title1}", f"### {title2}"]
        replace_env_table_in_md(md_path, combined, section_header=f"### {title1}", all_section_headers=all_section_headers)
        print(f"{title1}/{title2} env documentation updated in {md_path}")

if __name__ == "__main__":
    main()
