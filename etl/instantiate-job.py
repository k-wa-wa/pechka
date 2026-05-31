#!/usr/bin/env python3
import sys
import json
import subprocess
import argparse

def main():
    parser = argparse.ArgumentParser(description="Instantiate a Job from a placeholder in the cluster")
    parser.add_argument("--placeholder", required=True, help="Name of the placeholder Job")
    parser.add_argument("--name", required=True, help="Name of the new Job")
    parser.add_argument("--namespace", default="pechka", help="Kubernetes namespace")
    parser.add_argument("--image", help="Override container image")
    parser.add_argument("--image-pull-policy", help="Override image pull policy")
    parser.add_argument("--env", action="append", default=[], help="Set environment variable in KEY=VALUE format")
    parser.add_argument("--set-arg", action="append", default=[], help="Set container argument following a key, in KEY=VALUE format")

    args = parser.parse_args()

    # Get placeholder Job from the cluster
    cmd = ["kubectl", "get", "job", args.placeholder, "-n", args.namespace, "-o", "json"]
    try:
        data = subprocess.check_output(cmd)
    except subprocess.CalledProcessError as e:
        print(f"Error fetching placeholder job {args.placeholder}: {e}", file=sys.stderr)
        sys.exit(1)

    job = json.loads(data.decode("utf-8"))

    # Clean metadata
    metadata = job.get("metadata", {})
    metadata["name"] = args.name
    for key in ["uid", "resourceVersion", "creationTimestamp", "generation", "managedFields", "ownerReferences"]:
        metadata.pop(key, None)
    if "annotations" in metadata:
        metadata["annotations"].pop("batch.kubernetes.io/job-tracking", None)
        metadata["annotations"].pop("kubectl.kubernetes.io/last-applied-configuration", None)
    if "labels" in metadata:
        for lbl in ["controller-uid", "batch.kubernetes.io/controller-uid", "batch.kubernetes.io/job-name", "job-name"]:
            metadata["labels"].pop(lbl, None)

    # Clean spec
    spec = job.get("spec", {})
    spec.pop("selector", None)
    spec["suspend"] = False

    # Clean spec.template
    template = spec.get("template", {})
    template_meta = template.get("metadata", {})
    for key in ["creationTimestamp"]:
        template_meta.pop(key, None)
    if "labels" in template_meta:
        for lbl in ["controller-uid", "batch.kubernetes.io/controller-uid", "batch.kubernetes.io/job-name", "job-name"]:
            template_meta["labels"].pop(lbl, None)

    # Clean status
    job.pop("status", None)

    # Modify container spec
    if "spec" in template and "containers" in template["spec"] and template["spec"]["containers"]:
        container = template["spec"]["containers"][0]

        if args.image:
            container["image"] = args.image
        
        if args.image_pull_policy:
            container["imagePullPolicy"] = args.image_pull_policy

        if args.env:
            if "env" not in container or container["env"] is None:
                container["env"] = []
            for item in args.env:
                if "=" not in item:
                    continue
                k, v = item.split("=", 1)
                found = False
                for env_entry in container["env"]:
                    if env_entry.get("name") == k:
                        env_entry["value"] = v
                        env_entry.pop("valueFrom", None)
                        found = True
                        break
                if not found:
                    container["env"].append({"name": k, "value": v})

        if args.set_arg:
            if "args" in container and container["args"]:
                c_args = container["args"]
                for item in args.set_arg:
                    if "=" not in item:
                        continue
                    k, v = item.split("=", 1)
                    for i, arg_val in enumerate(c_args):
                        if arg_val == k and i + 1 < len(c_args):
                            c_args[i + 1] = v

    # Output JSON to stdout
    print(json.dumps(job, indent=2))

if __name__ == "__main__":
    main()
