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

The following environment variables (_ separated) or yaml keys in /etc/kube-dnstap.yaml can be set:

```yaml
suffixes:
  ignore:
    - .cluster.local.
  only: # Overrides ignore
    - .example.com.
