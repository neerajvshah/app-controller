# app-operator
You can run this operator against a local minikube cluster doing the following:
```
minikube start
make install
make run
```
The first command should set your `kubectl` commands to automatically target the local minikube cluster.

From there you can create a PodInfoRedisApplication (shortName `pira`) using:
```
kubectl apply -f ./hack/test-pira.yaml
```

A PodInfo service will be exposed through a NodePort - to tunnel this through to your computer's network, you can use:
```
minikube service whatever-podinfo --url
```
You can then navigate to that URL.

The above command will output the localhost URL the service is exposed on. If Redis is enabled, you can also send POST/PUT and GET requests to that same URL against `/cache/{key}` and verify connectivity between PodInfo and Redis. In the below example, I am using URL `http://127.0.0.1:56937`.
```
URL="http://127.0.0.1:56937"
curl -H 'Content-Type: application/json' -d "hello world" -X POST $URL/cache/test
curl $URL/cache/test
```

You should see your deployments and services come up. You can edit the CR and see changes flow through with:
```
kubectl edit pira whatever
```

You can delete the CR and see that all owned resources (deployments, services) will be automatically garbage collected:
```
kubeclt delete pira whatever
```

## Testing
Unit tests can be run with the following command:
```
make test
```

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

