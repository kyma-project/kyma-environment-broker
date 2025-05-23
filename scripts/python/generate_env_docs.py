#!/usr/bin/env python3
import re
import yaml

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

DEPLOYMENT_YAML = "resources/keb/templates/deployment.yaml"
VALUES_YAML = "resources/keb/values.yaml"
OUTPUT_MD = "docs/contributor/02-30-keb-configuration.md"

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
                        mval = re.search(r'value:\s*"?{{\s*\.Values\.([^"}}]+)\s*}}"?', val_line)
                        if mval:
                            env_vars.append((current_env, mval.group(1)))
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
        # Otherwise, leave desc and default blank (will render as '-')
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

def write_markdown_table(env_docs, output_path):
    """
    Write the environment variable documentation as a Markdown table.
    Fields with missing description or value are rendered as '-'.
    Long values in the first and second columns are split with a soft break (S\u200b;), except the value column, which is not split.
    The description column is visually wider.
    """
    with open(output_path, "w") as f:
        f.write("## Kyma Environment Broker Configuration\n\n")
        f.write("Kyma Environment Broker (KEB) binary allows you to override some configuration parameters. You can specify the following environment variables:\n\n")
        f.write("| Environment Variable | Current Value | Description |")
        f.write("\n|---------------------|------------------------------|---------------------------------------------------------------|\n")
        for doc in env_docs:
            desc = doc['description'] if doc['description'] else '-'
            # Format default value for Markdown
            if doc['default'] is None or doc['default'] == '':
                default = 'None'
            else:
                default = doc["default"]
            env_val = soft_break(doc["env"], 20, prefer_char='_')
            env_col = f'**{env_val}**'
            if default == 'None':
                val_col = 'None'
            else:
                # Do not split or break the value column at all
                val_col = f'<code>{str(default)}</code>'
            f.write(f"| {env_col} | {val_col} | {desc} |\n")

def main():
    """
    Main entry point: extract env vars, map to values.yaml, and write Markdown documentation.
    """
    env_vars = extract_env_vars_with_paths(DEPLOYMENT_YAML)
    values_doc = parse_values_yaml_with_comments(VALUES_YAML)
    env_docs = map_env_to_values(env_vars, values_doc)
    write_markdown_table(env_docs, OUTPUT_MD)
    print(f"Markdown documentation generated in {OUTPUT_MD}")

if __name__ == "__main__":
    main()
