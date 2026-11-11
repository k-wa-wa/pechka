#!/usr/bin/env python3
import os
import re
import sys
import yaml

var_pattern = re.compile(r"\$\(([A-Za-z0-9_]+)\)")
errors = []

for root, _, files in os.walk("k8s"):
    for f in files:
        if not (f.endswith(".yaml") or f.endswith(".yml")):
            continue
        filepath = os.path.join(root, f)
        with open(filepath, "r", encoding="utf-8") as fp:
            try:
                docs = list(yaml.load_all(fp, Loader=yaml.SafeLoader))
            except Exception:
                continue
        for doc in docs:
            if not isinstance(doc, dict):
                continue

            metadata = doc.get("metadata", {})
            annotations = metadata.get("annotations", {}) if isinstance(metadata, dict) else {}
            is_presync = False
            if isinstance(annotations, dict):
                hook = annotations.get("argocd.argoproj.io/hook", "")
                if "PreSync" in hook:
                    is_presync = True

            env_lists = []
            spec = doc.get("spec", {})
            if isinstance(spec, dict):
                template = spec.get("template", {})
                if isinstance(template, dict):
                    pod_spec = template.get("spec", {})
                    if isinstance(pod_spec, dict):
                        # PreSync フックを持つリソースが volumes で pechka-config を参照していないか検証
                        if is_presync:
                            for v in pod_spec.get("volumes", []):
                                if isinstance(v, dict):
                                    cm = v.get("configMap", {})
                                    if isinstance(cm, dict) and cm.get("name") == "pechka-config":
                                        errors.append(
                                            f"{filepath}: PreSync resource cannot mount 'pechka-config' (created in normal Sync phase)"
                                        )

                        for c in pod_spec.get("containers", []) + pod_spec.get("initContainers", []):
                            if isinstance(c, dict) and "env" in c and isinstance(c["env"], list):
                                env_lists.append((c.get("name", "container"), c["env"], is_presync))
                for t in spec.get("templates", []):
                    if isinstance(t, dict):
                        c = t.get("container", {})
                        if isinstance(c, dict) and "env" in c and isinstance(c["env"], list):
                            env_lists.append((t.get("name", "template"), c["env"], is_presync))
                        if "env" in t and isinstance(t["env"], list):
                            env_lists.append((t.get("name", "template"), t["env"], is_presync))

            for name, env, presync in env_lists:
                defined = set()
                for item in env:
                    if not isinstance(item, dict):
                        continue

                    # PreSync フックを持つリソースが pechka-config を参照していないか検証
                    if presync:
                        val_from = item.get("valueFrom", {})
                        if isinstance(val_from, dict):
                            cm_ref = val_from.get("configMapKeyRef", {})
                            if isinstance(cm_ref, dict) and cm_ref.get("name") == "pechka-config":
                                errors.append(
                                    f"{filepath} ({name}): PreSync resource cannot reference 'pechka-config' in env '{item.get('name')}'"
                                )

                    val = item.get("value")
                    var_name = item.get("name")
                    if isinstance(val, str):
                        for match in var_pattern.findall(val):
                            if match not in defined:
                                errors.append(f"{filepath} ({name}): $({match}) referenced in {var_name} before definition")
                    if var_name:
                        defined.add(var_name)

if errors:
    print("❌ Found issues in k8s manifests:")
    for err in errors:
        print("  " + err)
    sys.exit(1)

print("✅ All $(VAR) references and PreSync dependencies in k8s manifests passed verification.")
