#! /usr/bin/env python

import os
import subprocess
import requests
import json
import time

namespace = os.environ['NAMESPACE']
webhook = os.environ.get('WEBHOOK', None)

while True:
    p = subprocess.run(["./k8ecr", "deploy", namespace, "-"], stdout=subprocess.PIPE)
    if webhook is not None and p.stdout is not None:
        requests.post(
            webhook,
            data=json.dumps({'text': p.stdout.decode("utf-8")}),
            headers={'Content-Type': 'application/json'})
    time.sleep(60)
