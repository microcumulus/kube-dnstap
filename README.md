# kube-dnstap

Find out which pods are querying which domains in your Kubernetes cluster.

## Usage

```bash
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

