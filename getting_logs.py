import os
import subprocess
import json
import time
import argparse


def get_pods():
    cmd = "kubectl get pods -o json -n crossplane-system"
    process = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    output, error = process.communicate()
    pods = json.loads(output)
    return pods


def get_logs(pod_name):
    cmd = f"kubectl logs {pod_name} -n crossplane-system"
    process = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    output, error = process.communicate()
    return output.decode("utf-8")


def main():
    args = argparser.parse_args()
    tail = args.tail
    time_to_sleep = args.sleep
    provider_name = args.providername
    current_logs = ""
    while True:
        pods = get_pods()
        try:
            for pod in pods["items"]:
                pod_name = pod["metadata"]["name"]
                if pod_name.startswith(f"provider-{provider_name}"):
                    print(f"Getting logs for pod: {pod_name}")
                    logs = get_logs(pod_name)
                    # getting last tail lines
                    if tail == -1:
                        new_logs = logs
                    else:
                        logs = logs.split("\n")[-tail:]
                        new_logs = "\n".join(logs)
                    if new_logs != current_logs:
                        with open(f"provider_{provider_name}.log", "w") as f:
                            f.write(new_logs)
                        print(new_logs)
                        current_logs = new_logs
        except Exception as e:
            print(e)
            print("Error getting logs")
        time.sleep(time_to_sleep)


if __name__ == "__main__":
    # add arguments
    argparser = argparse.ArgumentParser()
    argparser.add_argument("--tail", help="logs to tail", default=-1, type=int)
    argparser.add_argument(
        "--sleep", help="time to sleep between each iteration", default=5, type=int
    )
    argparser.add_argument(
        "--providername",
        help="provider name",
        default="driftprovider",
        type=str,
    )
    main()
