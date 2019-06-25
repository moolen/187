# 187: exec into pod = death sentence
This is a admission webhook that kills a pod after spawning a shell using `kubectl exec`.

Why? Developers want to debug but we should reconcile the state of the pod.

See [./deploy](./deploy) for deployment manifests. TLS is mandatory for webhooks. The example uses `cert-manager` to generate and inject the TLS credentials into the webhook and the pod. The image is available via `moolen/187:latest`.

| env | default | description |
|--|--|--|
| `GRACE_PERIOD` | 15m | specify a grace period before killing the pod |
| `LOG_LEVEL` | info | set the log level|
| `TLS_CERT` | `` | path to the server certificate file |
| `TLS_KEY` | `` | path to the private key |
| `LISTEN` | `:8000` | port/address to listen on. It's always TLS |
