# kube-dnstap

Find out which pods are querying which domains in your Kubernetes cluster.

## Usage

```bash
$ kubectl create ns dnstap
$ kubectl apply -f https://raw.githubusercontent.com/microcumulus/kube-dnstap/master/k8s.yaml
```

CoreDNS:

```bash
$ kubectl -n kube-system edit cm coredns
```

Add to the main block:

```coredns
  dnstap {
    dnstap tcp://dnstap.default:12345 full
  }
```

## Configuration

The following configs can be set via environment variables (`_` separated) or
yaml keys in `/etc/kube-dnstap.yaml`:

```yaml
suffixes:
  ignore:
    - .cluster.local.
  only: # Overrides ignore
    - .example.com.
listen:
  addr: 0.0.0.0:12345
metrics:
  addr: 0.0.0.0:8080
```
