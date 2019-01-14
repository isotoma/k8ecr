#! /usr/bin/env python

import json
import os
import subprocess
import sys
import time

import requests

namespace = os.environ['NAMESPACE']
webhook = os.environ.get('WEBHOOK')

hookfile = sys.argv[1] if len(sys.argv) > 1 else None

while True:
    if hookfile:
        p = subprocess.run(["./k8ecr", "-w", hookfile, "deploy", namespace, "-"], stdout=subprocess.PIPE)
    else:
        p = subprocess.run(["./k8ecr", "deploy", namespace, "-"], stdout=subprocess.PIPE)
    if webhook is not None and p.stdout is not None:
        requests.post(
            webhook,
            data=json.dumps({'text': p.stdout.decode("utf-8")}),
            headers={'Content-Type': 'application/json'})
    time.sleep(60)
